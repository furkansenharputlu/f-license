package main

import (
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const license = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbG1hIjoiYXJtdXQiLCJ1c2VybmFtZSI6ImZ1cmthbiJ9.s8g9ldvCLQw7Pfy8TI9jbQHD7Hvn52tiTsuVRVLyZXM"
const secret = "test-secret"

var LicenseValid bool

func main() {
	cloud()
	//onpremise()
}

func onpremise() {
	for {
		token, err := jwt.Parse(license, func(token *jwt.Token) (interface{}, error) {
			// Don't forget to validate the alg is what you expect:
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return []byte(secret), nil
		})

		if err != nil {
			continue
		}

		if !token.Valid {
			LicenseValid = false
		} else {
			LicenseValid = true
		}

		fmt.Println(LicenseValid)
		time.Sleep(2 * time.Second)
	}
}

func cloud() {
	for {
		form := url.Values{}
		form.Add("license", license)
		resp, err := http.Post("http://localhost:4242/license/verify", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
		if err != nil {
			fmt.Println("License couldn't be verified:", err)
			continue
		}

		var res map[string]interface{}
		bytes, _ := ioutil.ReadAll(resp.Body)
		json.Unmarshal(bytes, &res)

		valid := res["valid"].(bool)

		if !valid {
			LicenseValid = false
		} else {
			LicenseValid = true
		}

		fmt.Println(LicenseValid)
		time.Sleep(2 * time.Second)
	}
}
