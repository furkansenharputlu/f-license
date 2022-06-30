package main

import (
	"crypto/tls"
	"fmt"
	"github.com/furkansenharputlu/f-license/config"
	"github.com/furkansenharputlu/f-license/lcs"
	"strings"

	"github.com/furkansenharputlu/f-license/storage"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"log"
	"net/http"
)

const Version = "0.1"

func intro() {
	logrus.Info("f-license ", Version)
	logrus.Info("Copyright Furkan Şenharputlu 2020")
	logrus.Info("https://f-license.com")
}

func setProductKeyIDs() {
	km := KeyManager{}
	var err error
	for _, product := range config.Global.Products {
		if (strings.HasPrefix(product.Alg, "HS") && product.Key.Type != "hmac") || (strings.HasPrefix(product.Alg, "RS") && product.Key.Type != "rsa") {
			logrus.Fatalf("alg and key type mismatch")
		}

		product.Key.ID, _, err = km.GetOrAddKey(product.Key, true)
		if err != nil {
			logrus.WithError(err).Fatalf("Error while calculating key ID of product: %s", product.Name)
		}
	}
}

func main() {

	intro()

	config.Global.Load("config.json")
	if !config.Global.LoadProductsFromDB {
		setProductKeyIDs()
	}

	storage.Connect(&lcs.License{}, &config.Key{}, &config.Product{})

	router := GenerateRouter()

	addr := fmt.Sprintf(":%d", config.Global.Port)
	certFile := config.Global.ServerOptions.CertFile
	keyFile := config.Global.ServerOptions.KeyFile

	if config.Global.ServerOptions.EnableTLS {
		srv := &http.Server{
			Addr:         addr,
			Handler:      router,
			TLSConfig:    &config.Global.ServerOptions.TLSConfig,
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
		}
		log.Fatal(srv.ListenAndServeTLS(certFile, keyFile))
	} else {
		log.Fatal(http.ListenAndServe(addr, router))
	}
}

func GenerateRouter() *mux.Router {
	r := mux.NewRouter()
	// Endpoints called by product owners
	adminRouter := r.PathPrefix("/f").Subrouter()
	adminRouter.Use(AuthenticationMiddleware)

	adminRouter.HandleFunc("/products", AddProduct).Methods(http.MethodPost)
	adminRouter.HandleFunc("/products", GetAllProducts).Methods(http.MethodGet)
	adminRouter.HandleFunc("/products/{id}", GetProduct).Methods(http.MethodGet)
	adminRouter.HandleFunc("/products/{id}", UpdateProduct).Methods(http.MethodPut)
	adminRouter.HandleFunc("/products/{id}", DeleteProduct).Methods(http.MethodDelete)

	adminRouter.HandleFunc("/licenses", GetAllLicenses).Methods(http.MethodGet)
	adminRouter.HandleFunc("/licenses", GenerateLicense).Methods(http.MethodPost)
	adminRouter.HandleFunc("/licenses/info", GetLicenseInfos).Methods(http.MethodGet)
	adminRouter.HandleFunc("/licenses/{id}", GetLicense).Methods(http.MethodGet)
	adminRouter.HandleFunc("/licenses/{id}/activate", ChangeLicenseActiveness).Methods(http.MethodPut)
	adminRouter.HandleFunc("/licenses/{id}/inactivate", ChangeLicenseActiveness).Methods(http.MethodPut)
	adminRouter.HandleFunc("/licenses/{id}", DeleteLicense).Methods(http.MethodDelete)

	adminRouter.HandleFunc("/keys", GetAllKeys).Methods(http.MethodGet)
	adminRouter.HandleFunc("/keys", UploadKey).Methods(http.MethodPost)
	adminRouter.HandleFunc("/keys/{id}", GetKey).Methods(http.MethodGet)

	adminRouter.HandleFunc("/keys/{id}", DeleteKey).Methods(http.MethodDelete)

	// Endpoints called by product instances having license
	r.HandleFunc("/license/verify", VerifyLicense).Methods(http.MethodPost)
	r.HandleFunc("/license/ping", Ping).Methods(http.MethodPost)

	return r
}
