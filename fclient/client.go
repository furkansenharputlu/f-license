package fclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
)

func VerifyRemotely(serverURL string, licenseKey string) (verified bool, err error) {
	form := url.Values{}
	form.Add("token", licenseKey)
	resp, err := http.Post(serverURL+"/license/verify", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}

	var res map[string]interface{}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	err = json.Unmarshal(bytes, &res)
	if err != nil {
		return false, err
	}

	return res["valid"].(bool), nil
}

func VerifyLocally(licenseKey string, publicKey string) (verified bool, err error) {
	if publicKey == "" {
		return false, errors.New("public key shouldn't be empty")
	}

	token, err := jwt.Parse(licenseKey, func(token *jwt.Token) (interface{}, error) {
		switch token.Method.(type) {
		case *jwt.SigningMethodHMAC:
			return []byte(publicKey), nil
		case *jwt.SigningMethodRSA:
			return jwt.ParseRSAPublicKeyFromPEM([]byte(publicKey))
		default:
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
	})

	return token.Valid, err
}
