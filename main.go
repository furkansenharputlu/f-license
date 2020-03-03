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
	"strings"
)

const Version = "0.1"

func intro() {
	logrus.Info("f-license ", Version)
	logrus.Info("Copyright Furkan Şenharputlu 2020")
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
	adminRouter.HandleFunc("/activate", ChangeLicenseActiveness).Methods(http.MethodPut)
	adminRouter.HandleFunc("/inactivate", ChangeLicenseActiveness).Methods(http.MethodPut)

	// Endpoints called by product instances having license
	r.HandleFunc("/license/verify", VerifyLicense).Methods(http.MethodPost)
	r.HandleFunc("/license/ping", Ping).Methods(http.MethodPost)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Global.Port), r))
}

type License struct {
	Type     string        `json:"type"`
	Hash     uint64        `json:"hash"`
	Claims   jwt.MapClaims `json:"claims"`
	Inactive bool          `json:"-"`
}

func authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != config.Global.AdminSecret {
			w.WriteHeader(http.StatusUnauthorized)
			ReturnResponse(w, map[string]interface{}{
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

func ChangeLicenseActiveness(w http.ResponseWriter, r *http.Request) {
	hash := r.FormValue("license_hash")
	var message string

	u, _ := strconv.ParseUint(hash, 10, 64)
	l, ok := licenses[u]
	if !ok {
		ReturnResponse(w, map[string]interface{}{
			"message": "license not found",
		})
		return
	}

	if strings.HasSuffix(r.URL.Path, "/inactivate") {
		l.Inactive = true
		message = "Inactivated"
		logrus.Infof(`License is successfully inactivated: %d`, u)
	} else {
		l.Inactive = false
		message = "Activated"
		logrus.Infof(`License is successfully activated: %d`, u)
	}

	ReturnResponse(w, map[string]interface{}{
		"message": message,
	})
}

func Ping(w http.ResponseWriter, r *http.Request) {

}

func VerifyLicense(w http.ResponseWriter, r *http.Request) {
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

	l, ok := licenses[h.Sum64()]
	if !ok {
		return false, fmt.Errorf("license not found")
	}

	if l.Inactive {
		return false, fmt.Errorf("license inactivated")
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
