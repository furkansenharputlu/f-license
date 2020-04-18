package client

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
)

func VerifyRemotely(serverURL string, cert string, licenseKey string) (verified bool, err error) {
	form := url.Values{}
	form.Add("token", licenseKey)

	request, _ := http.NewRequest(http.MethodPost, serverURL+"/license/verify", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(cert))

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	resp, err := client.Do(request)
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

	errMsg, ok := res["error"]
	if ok {
		return false, errors.New(errMsg.(string))
	}

	return res["valid"].(bool), nil
}

func VerifyLocally(publicKey string, licenseKey string) (verified bool, err error) {
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
