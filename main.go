package main

import (
	"context"
	"encoding/json"
	"f-license/config"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

const Version = "0.1"

func intro() {
	logrus.Info("f-license ", Version)
	logrus.Info("Copyright Furkan Şenharputlu 2020")
	logrus.Info("https://f-license.com")
}

var MongoClient *mongo.Client
var licensesCol *mongo.Collection

func main() {
	intro()

	config.Global.Load("config.json")

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	MongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		logrus.Fatalf("Problem while connecting to Mongo: %s", err)
	}

	licensesCol = MongoClient.Database("f-license").Collection("licenses")

	r := mux.NewRouter()
	// Endpoints called by product owners
	adminRouter := r.PathPrefix("/admin").Subrouter()
	adminRouter.Use(authenticationMiddleware)
	adminRouter.HandleFunc("/generate", GenerateLicense).Methods(http.MethodPost)
	adminRouter.HandleFunc("/{id}", GetLicense).Methods(http.MethodGet)
	adminRouter.HandleFunc("/{id}/activate", ChangeLicenseActiveness).Methods(http.MethodPut)
	adminRouter.HandleFunc("/{id}/inactivate", ChangeLicenseActiveness).Methods(http.MethodPut)

	// Endpoints called by product instances having license
	r.HandleFunc("/license/verify", VerifyLicense).Methods(http.MethodPost)
	r.HandleFunc("/license/ping", Ping).Methods(http.MethodPost)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Global.Port), r))
}

type License struct {
	ID     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Type   string             `bson:"type" json:"type"`
	Hash   string             `bson:"hash" json:"-"`
	Token  string             `bson:"token" json:"token"`
	Claims jwt.MapClaims      `bson:"claims" json:"claims"`
	Active bool               `bson:"active" json:"active"`
}

func authenticationMiddleware(next http.Handler) http.Handler {
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

func ReturnResponse(w http.ResponseWriter, statusCode int, resp map[string]interface{}) {
	bytes, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = fmt.Fprintf(w, string(bytes))
}

func ReturnError(w http.ResponseWriter, errMsg string) {
	logrus.Error(errMsg)
	resp := map[string]interface{}{
		"error": errMsg,
	}
	bytes, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, string(bytes))
}

func GetLicense(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	licenseID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		ReturnError(w, fmt.Sprintf("ID format error: %s", err))
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	filter := bson.M{"_id": licenseID}
	res := licensesCol.FindOne(ctx, filter)
	err = res.Err()
	if err != nil {
		ReturnError(w, fmt.Sprintf("error while getting license: %s", err))
		return
	}

	var l License
	_ = res.Decode(&l)

	ReturnResponse(w, 200, map[string]interface{}{
		"id":     l.ID,
		"type":   l.Type,
		"claims": l.Claims,
		"active": l.Active,
		"token":  l.Token,
	})
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

	l.Token = signedString

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	h := fnv.New64a()
	h.Write([]byte(signedString))
	l.Hash = fmt.Sprintf("%v", h.Sum64())

	filter := bson.M{"hash": l.Hash}
	res := licensesCol.FindOne(ctx, filter)
	err = res.Err()
	if err != nil {
		if err != mongo.ErrNoDocuments {
			ReturnError(w, fmt.Sprintf("error while checking the existence of license: %s", err))
			return
		}
	} else {
		var existingLicense License
		_ = res.Decode(&existingLicense)
		ReturnError(w, fmt.Sprintf("There is already such license: %s", existingLicense.ID.Hex()))
		return
	}

	l.ID = primitive.NewObjectID()

	update := bson.M{"$set": l}
	_, err = licensesCol.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		ReturnError(w, fmt.Sprintf("error while inserting license: %s", err))
		return
	}

	logrus.Info("License successfully generated")

	ReturnResponse(w, 200, map[string]interface{}{
		"id":      l.ID.Hex(),
		"license": l.Token,
	})
}

func ChangeLicenseActiveness(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	licenseID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		ReturnError(w, fmt.Sprintf("ID format error: %s", err))
		return
	}

	inactivate := strings.HasSuffix(r.URL.Path, "/inactivate")

	filter := bson.M{"_id": bson.M{"$eq": licenseID}}
	update := bson.M{"$set": bson.M{"active": !inactivate}}
	res, err := licensesCol.UpdateOne(context.Background(), filter, update)
	if res.MatchedCount == 0 {
		ReturnError(w, "There is no matching license")
		return
	}
	if err != nil {
		ReturnError(w, "License cannot be updated")
		return
	}

	var message string

	if inactivate {
		message = "Inactivated"
		logrus.Infof(`License is successfully inactivated: %s`, id)
	} else {
		message = "Activated"
		logrus.Infof(`License is successfully activated: %s`, id)
	}

	ReturnResponse(w, 200, map[string]interface{}{
		"message": message,
	})
}

func Ping(w http.ResponseWriter, r *http.Request) {

}

func VerifyLicense(w http.ResponseWriter, r *http.Request) {
	license := r.FormValue("license")
	ok, err := IsLicenseValid(license)
	if err != nil {
		ReturnResponse(w, http.StatusUnauthorized, map[string]interface{}{
			"valid":   false,
			"message": fmt.Sprintf("error while validating license: %s", err),
		})

		return
	}

	ReturnResponse(w, 200, map[string]interface{}{
		"valid": ok,
	})
}

func IsLicenseValid(license string) (bool, error) {
	h := fnv.New64a()
	h.Write([]byte(license))
	hash := h.Sum64()
	hashStr := fmt.Sprintf("%v", hash)

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	filter := bson.M{"hash": hashStr}
	res := licensesCol.FindOne(ctx, filter)
	err := res.Err()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, fmt.Errorf("license not found")
		}
		return false, fmt.Errorf("error while getting license: %s", err)
	}

	var l License
	_ = res.Decode(&l)

	if !l.Active {
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
