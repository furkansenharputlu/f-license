package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/furkansenharputlu/f-license/client"
	"github.com/furkansenharputlu/f-license/config"
	"github.com/furkansenharputlu/f-license/lcs"
	"github.com/furkansenharputlu/f-license/storage"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
)

var tr *TestRunner

type TestCase struct {
	Method     string
	Path       string
	Code       int
	Data       interface{}
	Headers    map[string]string
	FormParams map[string]string
	BodyMatch  string
}

type TestRunner struct {
	server *httptest.Server
	client *http.Client
}

func NewTestRunner() *TestRunner {
	return &TestRunner{httptest.NewServer(GenerateRouter()), &http.Client{}}
}

func TestMain(m *testing.M) {
	tr = NewTestRunner()
	config.Global.Load("sample_config.json")
	config.Global.DBName = "f-license_test"
	storage.Connect()
	_ = storage.LicenseHandler.DropDatabase()

	publicKeyFile, privateKeyFile := genKeys()
	defer func() {
		_ = privateKeyFile.Close()
		_ = publicKeyFile.Close()
	}()

	app := config.Global.Apps["test-app"]
	app.Signature.RSAPrivateKeyFile = privateKeyFile.Name()
	app.Signature.RSAPublicKeyFile = publicKeyFile.Name()
	app.Alg = "RS512"
	config.Global.Apps["test-app"] = app

	ret := m.Run()
	tr.server.Close()
	_ = storage.LicenseHandler.DropDatabase()
	os.Exit(ret)
}

func Reset() {
	ResetTestConfig()
	_ = storage.LicenseHandler.DropDatabase()
}

func ResetTestConfig() {
	app := config.Global.Apps["test-app"]
	app.Alg = "RS512"
	config.Global.Apps["test-app"] = app
}

func (tr *TestRunner) Run(t *testing.T, tc *TestCase) *http.Response {

	formParams := url.Values{}
	for k, v := range tc.FormParams {
		formParams.Add(k, v)
	}

	var reader io.Reader

	if len(formParams) != 0 {
		body := formParams.Encode()
		reader = strings.NewReader(body)
	} else {
		body, _ := json.Marshal(tc.Data)
		reader = bytes.NewReader(body)
	}

	r, err := http.NewRequest(tc.Method, tr.server.URL+tc.Path, reader)
	assert.NoError(t, err)

	r.Header.Set("Authorization", config.Global.AdminSecret)

	if len(formParams) != 0 {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := tr.client.Do(r)
	assert.NoError(t, err)

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	assert.NoError(t, err)

	if bodyMatch := regexp.MustCompile(tc.BodyMatch); !bodyMatch.MatchString(string(body)) {
		t.Fatalf("Response body does not match with regex `%s`. %s", bodyMatch, string(body))
	}

	return resp
}

func sampleLicense(lGen ...func(l *lcs.License)) (l *lcs.License) {
	l = &lcs.License{
		Active: true,
		Headers: map[string]interface{}{
			"typ": "Trial",
		},
		Claims: jwt.MapClaims{
			"name":    "Furkan",
			"address": "Istanbul, Turkey",
		},
	}

	if len(lGen) > 0 {
		lGen[0](l)
	}
	return
}

func TestClientVerifyLocally(t *testing.T) {
	defer Reset()
	t.Run("HS512", func(t *testing.T) {
		l := sampleLicense(func(l *lcs.License) {
			l.Headers["alg"] = "HS512"
		})

		_ = l.Generate()

		verified, _ := client.VerifyLocally("test-secret", l.Token)
		assert.True(t, verified)
	})

	t.Run("RS256", func(t *testing.T) {
		publicKeyFile, privateKeyFile := genKeys()
		defer func() {
			_ = privateKeyFile.Close()
			_ = publicKeyFile.Close()
		}()

		config.Global.DefaultSignature = config.Signature{
			RSAPrivateKeyFile: privateKeyFile.Name(),
			RSAPublicKeyFile:  publicKeyFile.Name(),
		}

		l := sampleLicense(func(l *lcs.License) {
			l.Headers["alg"] = "RS256"
		})
		_ = l.Generate()

		pkInBytes, _ := ioutil.ReadFile(publicKeyFile.Name())
		publicKey := string(pkInBytes)

		verified, _ := client.VerifyLocally(publicKey, l.Token)
		assert.True(t, verified)
	})
}

func TestClientVerifyRemotely(t *testing.T) {
	defer Reset()
	path := "/admin/licenses"

	t.Run("HS512", func(t *testing.T) {
		l := sampleLicense(func(l *lcs.License) {
			l.Headers["alg"] = "HS512"
		})

		resp := tr.Run(t, &TestCase{Method: http.MethodPost, Path: path, Data: l, BodyMatch: `"id":.*"token":"ey.*"`})
		resBytes, _ := ioutil.ReadAll(resp.Body)
		var resMap map[string]string
		_ = json.Unmarshal(resBytes, &resMap)

		// client code
		verified, _ := client.VerifyRemotely(tr.server.URL, "", resMap["token"])
		assert.True(t, verified)
	})

	t.Run("RS256", func(t *testing.T) {
		publicKeyFile, privateKeyFile := genKeys()
		defer func() {
			_ = privateKeyFile.Close()
			_ = publicKeyFile.Close()
		}()
		config.Global.DefaultSignature = config.Signature{
			RSAPrivateKeyFile: privateKeyFile.Name(),
			RSAPublicKeyFile:  publicKeyFile.Name(),
		}

		l := sampleLicense(func(l *lcs.License) {
			l.Headers["alg"] = "RS256"
		})
		_ = l.Generate()

		resp := tr.Run(t, &TestCase{Method: http.MethodPost, Path: path, Data: l, BodyMatch: `"id":.*"token":"ey.*"`})
		resBytes, _ := ioutil.ReadAll(resp.Body)
		var resMap map[string]string
		_ = json.Unmarshal(resBytes, &resMap)

		// client code
		verified, _ := client.VerifyRemotely(tr.server.URL, "", resMap["token"])
		assert.True(t, verified)
	})
}

func TestLicense_GetApp(t *testing.T) {
	l := sampleLicense()

	app, _ := l.GetApp("test-app")
	assert.Equal(t, config.Global.Apps["test-app"], app)

	_, err := l.GetApp("non-existing-app")
	assert.EqualError(t, err, "app not found with given name")

	_, err = l.GetApp("")
	assert.EqualError(t, err, "app not found with given name")
}

func TestLicense_ApplyApp(t *testing.T) {

	l := sampleLicense(func(l *lcs.License) {
		l.Headers["alg"] = "HS512"
	})

	t.Run("without applying", func(t *testing.T) {
		assert.Equal(t, "HS512", l.GetAlg())
	})

	t.Run("with applying", func(t *testing.T) {
		l.Headers["app"] = "test-app"
		_ = l.ApplyApp(l.GetAppName())

		assert.Equal(t, "RS512", l.GetAlg())
		assert.Equal(t, config.Global.Apps["test-app"].Alg, l.GetAlg())
	})
}

func genKeys() (publicKeyFile *os.File, privateKeyFile *os.File) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)
	template := &x509.Certificate{}
	template.SerialNumber = serialNumber
	template.BasicConstraintsValid = true
	template.NotBefore = time.Now()
	template.NotAfter = template.NotBefore.Add(time.Hour)

	derBytes, _ := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)

	var certPem bytes.Buffer
	pem.Encode(&certPem, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	publicKeyFile, _ = ioutil.TempFile("", "key.pem")
	_, _ = publicKeyFile.Write(certPem.Bytes())

	var keyPem bytes.Buffer
	_ = pem.Encode(&keyPem, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	privateKeyFile, _ = ioutil.TempFile("", "key.pem")
	_, _ = privateKeyFile.Write(keyPem.Bytes())

	return publicKeyFile, privateKeyFile
}
