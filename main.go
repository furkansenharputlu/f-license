package main

import (
	"encoding/json"
	"f-license/config"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

const Version = "0.1"

func intro() {
	logrus.Info("f-license ", Version)
	logrus.Info("Copyright Furkan Şenharputlu 2019")
	logrus.Info("https://f-license.com")
}

var licenses = make(map[uint64]*License)

func main() {
	intro()

	config.Global.Load("config.json")

	r := mux.NewRouter()
	// Endpoints called by product owners
	adminRouter := r.PathPrefix("/admin").Subrouter()
	adminRouter.Use(authenticationMiddleware)
	adminRouter.HandleFunc("/generate", GenerateLicense).Methods(http.MethodPost)
	adminRouter.HandleFunc("/inactivate", InactivateLicense).Methods(http.MethodPut)
	adminRouter.HandleFunc("/customer", CustomerHandler).Methods(http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete)

	// Endpoints called by product instances having license
	r.HandleFunc("/license/check", CheckLicense).Methods(http.MethodPost)
	r.HandleFunc("/license/ping", Ping).Methods(http.MethodPost)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Global.Port), r))
}

type License struct {
	Type     string        `json:"type"`
	Hash     uint64        `json:"hash"`
	Claims   jwt.MapClaims `json:"claims"`
	Inactive bool          `json:"-"`
}

type Customer struct {
	License License                `json:"license"`
	Details map[string]interface{} `json:"details"`
}

func authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != config.Global.AdminSecret {
			ReturnResponse(w, map[string]interface{}{
				"status":  http.StatusUnauthorized,
				"message": "Authorization failed",
			})
			return
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

func ReturnResponse(w http.ResponseWriter, resp map[string]interface{}) {
	bytes, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, string(bytes))
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
	h := fnv.New64a()
	h.Write([]byte(signedString))

	hash := h.Sum64()

	licenses[hash] = &l

	ReturnResponse(w, map[string]interface{}{
		"license":      signedString,
		"license_hash": hash,
	})
}

func InactivateLicense(w http.ResponseWriter, r *http.Request) {
	hash := r.FormValue("license_hash")

	u, _ := strconv.ParseUint(hash, 10, 64)
	licenses[u].Inactive = true

	logrus.Infof(`License is successfully inactivated: %d`, u)
	ReturnResponse(w, map[string]interface{}{
		"message": "Inactivated",
	})
}

func Ping(w http.ResponseWriter, r *http.Request) {

}

func CheckLicense(w http.ResponseWriter, r *http.Request) {
	license := r.FormValue("license")
	ok, err := IsLicenseValid(license)
	if err != nil {
		ReturnResponse(w, map[string]interface{}{
			"valid":   false,
			"message": fmt.Sprintf("error while validating license: %s", err),
		})

		return
	}

	ReturnResponse(w, map[string]interface{}{
		"valid": ok,
	})
}

func IsLicenseValid(license string) (bool, error) {
	h := fnv.New64a()
	h.Write([]byte(license))

	if licenses[h.Sum64()].Inactive {
		return false, fmt.Errorf("inactivated")
	}

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
