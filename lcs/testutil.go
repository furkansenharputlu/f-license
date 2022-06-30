package lcs

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/furkansenharputlu/f-license/config"
	"io/ioutil"
	"math/big"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
)

const (
	TestHMACSecret        = "test-hmac-secret"
	TestProductHMACSecret     = "test-product-hmac-secret"
	TestDefaultHMACSecret = "test-default-hmac-secret"

	TestProductName = "test-product"
)

func ResetTestConfig() {
	product := config.Global.Products["test-product"]
	product.Alg = "RS512"
	config.Global.Products["test-product"] = product
}

func SampleLicense(lGen ...func(l *License)) (l *License) {
	publicKeyFile, privateKeyFile := SampleKeys()
	defer func() {
		_ = privateKeyFile.Close()
		_ = publicKeyFile.Close()
	}()

	l = &License{
		LicenseInfo: LicenseInfo{
			Active: true,
			Headers: map[string]interface{}{
				"typ": "Trial",
				"alg": "HS256",
			},
			Claims: jwt.MapClaims{
				"name":    "Furkan",
				"address": "Istanbul, Turkey",
			},
		},
		Key:  &config.Key{
			/*HMAC: &config.KeyDetail{ // TODO gorm
				Raw: TestHMACSecret,
			},
			RSA: &config.RSA{
				Private: &config.KeyDetail{
					FilePath: privateKeyFile.Name(),
				},
				Public: &config.KeyDetail{
					FilePath: publicKeyFile.Name(),
				},
			},*/
		},
	}

	if len(lGen) > 0 {
		lGen[0](l)
	}
	return
}

func SampleProduct() {
	publicKeyFile, privateKeyFile := SampleKeys()
	defer func() {
		_ = privateKeyFile.Close()
		_ = publicKeyFile.Close()
	}()
	product := config.Global.Products["test-product"]
	/*product.Key.HMAC = &config.KeyDetail{ // TODO gorm
		Raw: TestProductHMACSecret,
	}
	product.Key.RSA.Private.FilePath = privateKeyFile.Name()
	product.Key.RSA.Public.FilePath = publicKeyFile.Name()
	product.Alg = "RS512"*/
	config.Global.Products["test-product"] = product
}

func SampleKeys() (publicKeyFile *os.File, privateKeyFile *os.File) {
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
