package main

import (
	"f-license/config"
	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestIsLicenseValid(t *testing.T) {
	config.Global.Secret = "test-secret"

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

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	license, err := token.SignedString([]byte(config.Global.Secret))
	if err != nil {
		logrus.Error(err)
	}
	valid, _ := IsLicenseValid(license)

	if !valid {
		t.Errorf("License is invalid")
	}
}
