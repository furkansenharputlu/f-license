package main

import (
	"crypto/rsa"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
	"io/ioutil"
)

const privKeyPath = "private.pem"
const pubKeyPath = "public.pem"

var (
	verifyKey *rsa.PublicKey
	signKey   *rsa.PrivateKey
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

func main() {
	// Server will be implemented.
}

func CheckLicenseValid(license string) bool {
	token, err := jwt.Parse(license, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return verifyKey, nil
	})

	if err != nil {
		logrus.Error(err)
		return false
	}

	if !token.Valid {
		return false
	}

	return true
}
