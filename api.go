package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	"io/ioutil"
	"net/http"
	"strings"

	"github.com/furkansenharputlu/f-license/config"
	"github.com/furkansenharputlu/f-license/lcs"
	"github.com/furkansenharputlu/f-license/storage"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func isSHA256(value string) bool {
	// check if hex encoded
	if _, err := hex.DecodeString(value); err != nil {
		return false
	}

	return true
}

func UploadKey(w http.ResponseWriter, r *http.Request) {

	_ = r.ParseMultipartForm(0)

	newKey := &config.Key{
		Name: "myRSAPair", //r.FormValue("name"), TODO: Enable this
	}

	typ := r.FormValue("type")
	if typ == "rsa" {
		var rsaPrivate, rsaPublic string
		if rsaPrivate = r.FormValue("rsaPrivateRaw"); rsaPrivate == "" {
			rsaPrivateFile, _, err := r.FormFile("rsaPrivateFile")
			if err != nil {
				ReturnError(w, http.StatusBadRequest, err.Error())
				return
			}

			rsaPrivateBytes, _ := ioutil.ReadAll(rsaPrivateFile)
			rsaPrivate = string(rsaPrivateBytes)
		}

		if rsaPublic = r.FormValue("rsaPublicRaw"); rsaPublic == "" {
			rsaPublicFile, _, err := r.FormFile("rsaPublicFile")
			if err != nil {
				ReturnError(w, http.StatusBadRequest, err.Error())
				return
			}

			rsaPublicBytes, _ := ioutil.ReadAll(rsaPublicFile)
			rsaPublic = string(rsaPublicBytes)
		}

		newKey.Type = "rsa"
		newKey.RSA = &config.RSA{
			Private: &config.KeyDetail{
				Raw: rsaPrivate,
			},
			Public: &config.KeyDetail{
				Raw: rsaPublic,
			},
		}
	} else if typ == "hmac" {
		var hmac string
		if hmac = r.FormValue("hmacRaw"); hmac == "" {
			hmacFile, _, err := r.FormFile("hmacFile")
			if err != nil {
				ReturnError(w, http.StatusBadRequest, err.Error())
				return
			}

			hmacBytes, _ := ioutil.ReadAll(hmacFile)
			hmac = string(hmacBytes)
		}

		newKey.Type = "hmac"
		newKey.HMAC = &config.KeyDetail{
			Raw: hmac,
		}
	} else {
		ReturnError(w, http.StatusBadRequest, "unknown type")
		return
	}

	var km KeyManager
	id, statusCode, err := km.GetOrAddKey(newKey, false)
	if err != nil {
		ReturnError(w, statusCode, err.Error())
		return
	}

	ReturnResponse(w, http.StatusOK, map[string]interface{}{
		"id": id,
	})

}

func GetAllKeys(w http.ResponseWriter, r *http.Request) {
	keys := make([]*config.Key, 0)
	err := storage.GlobalKeyHandler.GetAll(&keys)
	if err != nil {
		ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ReturnResponse(w, 200, keys)
}

func GetKey(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	km := KeyManager{}
	key := &config.Key{ID: id}
	_, statusCode, err := km.GetOrAddKey(key, false)
	if err != nil {
		ReturnResponse(w, statusCode, err.Error())
		return
	}

	ReturnResponse(w, http.StatusOK, &key)
}

func DeleteKey(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	err := storage.GlobalKeyHandler.DeleteByID(id)
	if err != nil {
		logrus.WithError(err).Error("Error while deleting key")
		ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ReturnResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Key successfully deleted",
	})
}

func LoadKey(l *lcs.License) error {
	km := KeyManager{}

	_, _, err := km.GetOrAddKey(&l.Key, false) // TODO: Handle status code
	if err != nil {
		return err
	}

	if strings.HasPrefix(l.GetAlg(), "HS") {
		if l.Key.RSA != nil {
			return errors.New("alg and key type mismatch")
		}

		l.SignKey = []byte(l.Key.HMAC.Raw)
		l.VerifyKey = []byte(l.Key.HMAC.Raw)
		l.Key.HMAC = nil // To prevent saving key in license
	} else {
		if l.Key.HMAC != nil {
			return errors.New("alg and key type mismatch")
		}
		l.VerifyKey, err = jwt.ParseRSAPublicKeyFromPEM([]byte(l.Key.RSA.Public.Raw))
		if err != nil {
			return err
		}

		l.SignKey, err = jwt.ParseRSAPrivateKeyFromPEM([]byte(l.Key.RSA.Private.Raw))
		if err != nil {
			return err
		}

		l.Key.RSA = nil // To prevent saving key in license
	}

	return nil
}

func GenerateLicense(w http.ResponseWriter, r *http.Request) {
	bytes, _ := ioutil.ReadAll(r.Body)

	var l lcs.License
	err := json.Unmarshal(bytes, &l)
	if err != nil {
		logrus.WithError(err).Error("Request body couldn't be marshalled")
		ReturnError(w, http.StatusBadRequest, err.Error())
		return
	}

	_ = l.ApplyApp() //TODO: Handle error

	err = LoadKey(&l)
	if err != nil {
		logrus.WithError(err).Error("Keys couldn't be loaded")
		ReturnError(w, http.StatusBadRequest, err.Error())
		return
	}

	err = l.Generate()
	if err != nil {
		logrus.WithError(err).Error("License couldn't be generated")
		ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	err, errCode := storage.LicenseHandler.AddIfNotExisting(&l)
	if err != nil {
		logrus.WithError(err).Error("License couldn't be stored")

		responseCode := http.StatusInternalServerError

		if errCode == storage.ItemDuplicationError {
			responseCode = http.StatusConflict
		}

		ReturnError(w, responseCode, err.Error())
		return
	}

	ReturnResponse(w, http.StatusOK, map[string]interface{}{
		"id":    l.ID,
		"token": l.Token,
	})
}

func GetApp(w http.ResponseWriter, r *http.Request) {
	//appName := mux.Vars(r)["name"]

	var app = config.App{
		Name: "test-app",
		Alg:  "HS512",
		Key: config.Key{
			HMAC: &config.KeyDetail{
				Raw: "test-secret",
			},
		},
	}

	ReturnResponse(w, 200, app)
}

func GetAllApps(w http.ResponseWriter, r *http.Request) {

	ReturnResponse(w, http.StatusOK, config.Global.Apps)
}

func GetLicense(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var l lcs.License
	err := storage.LicenseHandler.GetByID(id, &l)
	if err != nil {
		ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ReturnResponse(w, 200, &l)
}

func GetAllLicenses(w http.ResponseWriter, r *http.Request) {
	licenses := make([]*lcs.License, 0)
	err := storage.LicenseHandler.GetAll(&licenses)
	if err != nil {
		ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ReturnResponse(w, 200, licenses)
}

func ChangeLicenseActiveness(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	inactivate := strings.Contains(r.URL.Path, "/inactivate")

	err := storage.LicenseHandler.Activate(id, inactivate)
	if err != nil {
		logrus.WithError(err).Error("Error while activeness change")
		ReturnError(w, http.StatusInternalServerError, err.Error())
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
	token := r.FormValue("token")

	var l lcs.License
	err := storage.LicenseHandler.GetByToken(token, &l)
	if err != nil {
		logrus.WithError(err).Error("Error while getting license")
		ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	err = l.ApplyApp()
	if err != nil {
		ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	//LoadVerifyKey(&l)

	ok, err := l.IsLicenseValid(token)
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

	err := storage.LicenseHandler.DeleteByID(id)
	if err != nil {
		logrus.WithError(err).Error("Error while deleting license")
		ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ReturnResponse(w, 200, map[string]interface{}{
		"message": "License successfully deleted",
	})
}

func Ping(w http.ResponseWriter, r *http.Request) {

}

func ReturnResponse(w http.ResponseWriter, statusCode int, resp interface{}) {
	bytes, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = fmt.Fprintf(w, string(bytes))
}

func ReturnError(w http.ResponseWriter, statusCode int, errMsg string) {
	resp := map[string]interface{}{
		"error": errMsg,
	}
	bytes, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = fmt.Fprintf(w, string(bytes))
}

func AuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != config.Global.ControlAPISecret {
			ReturnResponse(w, http.StatusUnauthorized, map[string]interface{}{
				"message": "Authorization failed",
			})
			return
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}
