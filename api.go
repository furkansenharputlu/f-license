package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/furkansenharputlu/f-license/config"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io/ioutil"
	"net/http"
	"strings"

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
		Name: r.FormValue("name"),
	}

	switch r.FormValue("type") {
	case "rsa":
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
		newKey.Private = rsaPrivate
		newKey.Public = rsaPublic
	case "hmac":
		var secret string
		if secret = r.FormValue("hmacRaw"); secret == "" {
			hmacFile, _, err := r.FormFile("hmacFile")
			if err != nil {
				ReturnError(w, http.StatusBadRequest, err.Error())
				return
			}

			hmacBytes, _ := ioutil.ReadAll(hmacFile)
			secret = string(hmacBytes)
		}

		newKey.Type = "hmac"
		newKey.HMAC = secret
	default:
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
	keys := make([]*config.KeyInfo, 0)
	err := storage.SQLHandler.GetAll(&keys)
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

	err := storage.SQLHandler.Delete(&config.Key{}, "id = ?", id)
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

	l.Key = &config.Key{
		ID: l.KeyID,
	}

	_, _, err := km.GetOrAddKey(l.Key, false) // TODO: Handle status code
	if err != nil {
		return err
	}

	switch l.Key.Type {
	case "hmac":
		l.SignKey = []byte(l.Key.HMAC)
		l.VerifyKey = []byte(l.Key.HMAC)
		l.Key.Private = ""
		l.Key.Public = ""
	case "rsa":
		/*if l.Key.HMAC != nil { // TODO gorm
			return errors.New("alg and key type mismatch")
		}*/

		l.VerifyKey, err = jwt.ParseRSAPublicKeyFromPEM([]byte(l.Key.Public))
		if err != nil {
			return err
		}

		l.SignKey, err = jwt.ParseRSAPrivateKeyFromPEM([]byte(l.Key.Private))
		if err != nil {
			return err
		}

		l.Key.Public = "" // To prevent saving key in license
		l.Key.Private = ""
	default:
		return fmt.Errorf("unknown key type: %s", l.Key.Type)
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

	applyProduct := mux.Vars(r)["apply_product"]
	if applyProduct == "true" {
		_ = l.ApplyProduct() //TODO: Handle error
	}

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

	err = storage.SQLHandler.AddIfNotExisting(&l)
	if err != nil {
		logrus.WithError(err).Error("License couldn't be stored")

		responseCode := http.StatusInternalServerError

		ReturnError(w, responseCode, err.Error())
		return
	}

	ReturnResponse(w, http.StatusOK, map[string]interface{}{
		"id":    l.ID,
		"token": l.Token,
	})
}

func GetProduct(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var product config.Product
	err := storage.SQLHandler.Get(&product, "id = ?", id)
	if err != nil {
		ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ReturnResponse(w, 200, product)
}

func AddProduct(w http.ResponseWriter, r *http.Request) {
	bytes, _ := ioutil.ReadAll(r.Body)

	var p config.Product
	_ = json.Unmarshal(bytes, &p)

	p.ID = primitive.NewObjectID().Hex()

	err := storage.SQLHandler.AddIfNotExisting(&p)
	if err != nil {
		logrus.WithError(err).Error("Customer couldn't be stored")
		ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ReturnResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Product created successfully",
		"id":      p.ID,
	})
}

func UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	bytes, _ := ioutil.ReadAll(r.Body)

	var p config.Product
	_ = json.Unmarshal(bytes, &p)

	p.ID = id

	err := storage.SQLHandler.Update(&p, &p)
	if err != nil {
		logrus.WithError(err).Error("Error while updating product")
		ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ReturnResponse(w, 200, map[string]interface{}{
		"message": "Product successfully updated",
	})
}

func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	err := storage.SQLHandler.Delete(&config.Product{}, "id = ?", id)
	if err != nil {
		logrus.WithError(err).Error("Error while deleting product")
		ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ReturnResponse(w, 200, map[string]interface{}{
		"message": "Product successfully deleted",
	})
}

func GetAllProducts(w http.ResponseWriter, r *http.Request) {
	var products []config.Product
	if !config.Global.LoadProductsFromDB {
		for name, product := range config.Global.Products {
			product.Name = name
			products = append(products, product)
		}

		ReturnResponse(w, 200, products)
		return
	}

	err := storage.SQLHandler.GetAll(&products)
	if err != nil {
		ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ReturnResponse(w, http.StatusOK, products)
}

func GetLicense(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var l lcs.License
	err := storage.SQLHandler.Get(&l, "id = ?", id)
	if err != nil {
		ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	//l.DecodeToken()

	ReturnResponse(w, http.StatusOK, &l)
}

func GetAllLicenses(w http.ResponseWriter, r *http.Request) {
	licenses := make([]*lcs.License, 0)
	err := storage.SQLHandler.GetAll(&licenses)
	if err != nil {
		ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ReturnResponse(w, http.StatusOK, licenses)
}

func GetLicenseInfos(w http.ResponseWriter, r *http.Request) {
	customerID := r.FormValue("customerId")

	licenses := make([]*lcs.LicenseInfo, 0)
	err := storage.SQLHandler.Get(&licenses, "id = ? AND customerId = ?", customerID)
	if err != nil {
		ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ReturnResponse(w, http.StatusOK, licenses)
}

func ChangeLicenseActiveness(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	inactivate := strings.Contains(r.URL.Path, "/inactivate")

	err := storage.SQLHandler.Activate(id, inactivate)
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
	err := storage.SQLHandler.Get(token, &l) // TODO
	if err != nil {
		logrus.WithError(err).Error("Error while getting license")
		ReturnError(w, http.StatusInternalServerError, err.Error())
		return
	}

	err = l.ApplyProduct()
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

	err := storage.SQLHandler.Delete(&lcs.License{}, "id = ?", id)
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
