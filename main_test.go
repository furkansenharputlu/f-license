package main

import (
	"bytes"

	"encoding/json"

	"io"
	"io/ioutil"

	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/furkansenharputlu/f-license/client"
	"github.com/furkansenharputlu/f-license/config"
	"github.com/furkansenharputlu/f-license/lcs"
	"github.com/furkansenharputlu/f-license/storage"

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

	lcs.SampleApp()

	ret := m.Run()
	tr.server.Close()
	_ = storage.LicenseHandler.DropDatabase()
	os.Exit(ret)
}

func Reset() {
	lcs.ResetTestConfig()
	_ = storage.LicenseHandler.DropDatabase()
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

	r.Header.Set("Authorization", config.Global.ControlAPISecret)

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

func TestClientVerifyLocally(t *testing.T) {
	defer Reset()
	t.Run("HS512", func(t *testing.T) {
		l := lcs.SampleLicense(func(l *lcs.License) {
			l.Headers["alg"] = "HS512"
		})

		LoadKey(l)

		_ = l.Generate()

		verified, _ := client.VerifyLocally(lcs.TestHMACSecret, l.Token)
		assert.True(t, verified)
	})

	t.Run("RS256", func(t *testing.T) {
		l := lcs.SampleLicense(func(l *lcs.License) {
			l.Headers["alg"] = "RS256"
		})

		LoadRSAPair(l)
		_ = l.Generate()

		pkInBytes, _ := ioutil.ReadFile(l.Keys.RSA.Public.FilePath)
		publicKey := string(pkInBytes)

		verified, _ := client.VerifyLocally(publicKey, l.Token)
		assert.True(t, verified)
	})
}

func TestClientVerifyRemotely(t *testing.T) {
	defer Reset()
	path := "/admin/licenses"

	t.Run("HS512", func(t *testing.T) {
		l := lcs.SampleLicense(func(l *lcs.License) {
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
		publicKeyFile, privateKeyFile := lcs.SampleKeys()
		defer func() {
			_ = privateKeyFile.Close()
			_ = publicKeyFile.Close()
		}()
		config.Global.DefaultKeys = config.Keys{
			RSA: config.RSA{
				Private: config.Key{
					FilePath: privateKeyFile.Name(),
				},
				Public: config.Key{
					FilePath: publicKeyFile.Name(),
				},
			},
		}

		l := lcs.SampleLicense(func(l *lcs.License) {
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
	l := lcs.SampleLicense()

	app, _ := l.GetApp("test-app")
	assert.Equal(t, config.Global.Apps["test-app"], app)

	_, err := l.GetApp("non-existing-app")
	assert.EqualError(t, err, "app not found with given name")

	_, err = l.GetApp("")
	assert.EqualError(t, err, "app not found with given name")
}

func TestLicense_ApplyApp(t *testing.T) {

	l := lcs.SampleLicense(func(l *lcs.License) {
		l.Headers["alg"] = "HS512"
	})

	t.Run("without applying", func(t *testing.T) {
		assert.Equal(t, "HS512", l.GetAlg())
	})

	t.Run("with applying", func(t *testing.T) {
		l.Headers["app"] = "test-app"
		_ = l.ApplyApp()

		assert.Equal(t, "RS512", l.GetAlg())
		assert.Equal(t, config.Global.Apps["test-app"].Alg, l.GetAlg())
	})
}
