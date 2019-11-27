package main

import (
	"encoding/json"
	"f-license/config"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"log"
	"net/http"
)

const Version = "0.1"

func intro() {
	logrus.Info("f-license ", Version)
	logrus.Info("Copyright Furkan Şenharputlu 2019")
	logrus.Info("https://f-license.com")
}

func main() {
	intro()

	config.Global.Load("config.json")

	router := mux.NewRouter()
	// Endpoints called by product owners
	router.HandleFunc("/generate", GenerateLicense).Methods(http.MethodPost)
	router.HandleFunc("/customer", CustomerHandler).Methods(http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete)

	// Endpoints called by product instances having license
	router.HandleFunc("/license/check", CheckLicense).Methods(http.MethodPost)
	router.HandleFunc("/license/ping", Ping).Methods(http.MethodPost)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Global.Port), router))
}

type License struct {
	Type   string        `json:"type"`
	Claims jwt.MapClaims `json:"claims"`
}

type Customer struct {
	License License                `json:"license"`
	Details map[string]interface{} `json:"details"`
}

func CustomerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:

	case http.MethodPost:
		bytes, _ := ioutil.ReadAll(r.Body)

		c := Customer{}
		_ = json.Unmarshal(bytes, &c)

		fmt.Println(c)
	case http.MethodPut:

	case http.MethodDelete:

	}
}

func GenerateLicense(w http.ResponseWriter, r *http.Request) {
	bytes, _ := ioutil.ReadAll(r.Body)

	var l License
	_ = json.Unmarshal(bytes, &l)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, l.Claims)
	signedString, err := token.SignedString([]byte(config.Global.Secret))
	if err != nil {
		logrus.Error("Error signing token:", err)
	}

	logrus.Info("License successfully generated")

	_, _ = fmt.Fprintf(w, signedString)
}

func Ping(w http.ResponseWriter, r *http.Request) {

}

func CheckLicense(w http.ResponseWriter, r *http.Request) {
	license := r.FormValue("license")
	ok, err := IsLicenseValid(license)
	if err != nil {
		_, _ = fmt.Fprintf(w, "error while parsing license: %s", err)
		return
	}

	if ok {
		_, _ = fmt.Fprintf(w, "valid")
	} else {
		_, _ = fmt.Fprintf(w, "invalid")
	}
}

func IsLicenseValid(license string) (bool, error) {
	token, err := jwt.Parse(license, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(config.Global.Secret), nil
	})

	if err != nil {
		logrus.Error(err)
		return false, err
	}

	if !token.Valid {
		return false, nil
	}

	return true, nil
}
