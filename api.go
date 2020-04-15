package main

import (
	"encoding/json"
	"f-license/config"
	"f-license/lcs"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strings"
)

func GenerateLicense(w http.ResponseWriter, r *http.Request) {
	bytes, _ := ioutil.ReadAll(r.Body)

	var l lcs.License
	_ = json.Unmarshal(bytes, &l)

	err := l.Add()
	if err != nil {
		logrus.WithError(err).Error("Error while generating license")
		ReturnError(w, err.Error())
		return
	}

	ReturnResponse(w, 200, map[string]interface{}{
		"id":    l.ID.Hex(),
		"token": l.Token,
	})
}

func GetLicense(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var l lcs.License
	err := l.GetByID(id)
	if err != nil {
		ReturnError(w, err.Error())
		return
	}

	ReturnResponse(w, 200, map[string]interface{}{
		"id":     l.ID,
		"type":   l.Type,
		"claims": l.Claims,
		"active": l.Active,
		"token":  l.Token,
	})
}

func ChangeLicenseActiveness(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	inactivate := strings.Contains(r.URL.Path, "/inactivate")

	var l lcs.License
	err := l.Activate(id, inactivate)
	if err != nil {
		logrus.WithError(err).Error("Error while activeness change")
		ReturnError(w, err.Error())
		return
	}

	var message string

	if inactivate {
		message = "Inactivated"
	} else {
		message = "Activated"
	}

	ReturnResponse(w, 200, map[string]interface{}{
		"message": message,
	})
}

func VerifyLicense(w http.ResponseWriter, r *http.Request) {
	license := r.FormValue("license")

	var l lcs.License
	err := l.GetByToken(license)
	if err != nil {
		logrus.WithError(err).Error("Error while getting license")
		ReturnError(w, err.Error())
		return
	}

	ok, err := l.IsLicenseValid(license)
	if err != nil {
		ReturnResponse(w, http.StatusUnauthorized, map[string]interface{}{
			"valid":   false,
			"message": err.Error(),
		})

		return
	}

	ReturnResponse(w, 200, map[string]interface{}{
		"valid": ok,
	})
}

func DeleteLicense(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var l lcs.License
	err := l.DeleteByID(id)
	if err != nil {
		logrus.WithError(err).Error("Error while deleting license")
		ReturnError(w, err.Error())
		return
	}

	ReturnResponse(w, 200, map[string]interface{}{
		"message": "License successfully deleted",
	})
}

func Ping(w http.ResponseWriter, r *http.Request) {

}

func ReturnResponse(w http.ResponseWriter, statusCode int, resp map[string]interface{}) {
	bytes, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = fmt.Fprintf(w, string(bytes))
}

func ReturnError(w http.ResponseWriter, errMsg string) {
	resp := map[string]interface{}{
		"error": errMsg,
	}
	bytes, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, string(bytes))
}

func AuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != config.Global.AdminSecret {
			ReturnResponse(w, http.StatusUnauthorized, map[string]interface{}{
				"message": "Authorization failed",
			})
			return
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}
