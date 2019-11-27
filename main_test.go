package main

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"testing"
)

// read the key files before starting http handlers
func init() {
	signBytes, err := ioutil.ReadFile(privKeyPath)
	if err != nil {
		logrus.Fatal(err)
	}

	signKey, err = jwt.ParseRSAPrivateKeyFromPEM(signBytes)
	if err != nil {
		logrus.Fatal(err)
	}

	verifyBytes, err := ioutil.ReadFile(pubKeyPath)
	if err != nil {
		logrus.Fatal(err)
	}

	verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
	if err != nil {
		logrus.Fatal(err)
	}
}

func TestCheckLicenseValid(t *testing.T) {
	type MyCustomClaims struct {
		Foo string `json:"foo"`
		jwt.StandardClaims
	}

	// Create the Claims
	claims := MyCustomClaims{
		"bar",
		jwt.StandardClaims{
			ExpiresAt: 0,
			Issuer:    "test",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	license, err := token.SignedString(signKey)
	if err != nil {
		logrus.Error(err)
	}

	if !CheckLicenseValid(license) {
		t.Errorf("License was valid")
	}
}
