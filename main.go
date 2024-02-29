package main

import (
	"crypto/tls"
	"fmt"

	"github.com/furkansenharputlu/f-license/config"
	"github.com/furkansenharputlu/f-license/storage"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"log"
	"net/http"
)

const Version = "0.1"

func intro() {
	logrus.Info("f-license ", Version)
	logrus.Info("Copyright Furkan Şenharputlu 2024")
	logrus.Info("https://f-license.com")
}

func main() {

	intro()

	config.Global.Load("config.json")
	storage.Connect()

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
	adminRouter := r.PathPrefix("/admin").Subrouter()
	adminRouter.Use(AuthenticationMiddleware)
	adminRouter.HandleFunc("/licenses", GetAllLicenses).Methods(http.MethodGet)
	adminRouter.HandleFunc("/licenses", GenerateLicense).Methods(http.MethodPost)
	adminRouter.HandleFunc("/licenses/{id}", GetLicense).Methods(http.MethodGet)
	adminRouter.HandleFunc("/licenses/{id}/activate", ChangeLicenseActiveness).Methods(http.MethodPut)
	adminRouter.HandleFunc("/licenses/{id}/inactivate", ChangeLicenseActiveness).Methods(http.MethodPut)
	adminRouter.HandleFunc("/licenses/{id}/delete", DeleteLicense).Methods(http.MethodDelete)

	// Endpoints called by product instances having license
	r.HandleFunc("/license/verify", VerifyLicense).Methods(http.MethodPost)
	r.HandleFunc("/license/ping", Ping).Methods(http.MethodPost)

	return r
}
