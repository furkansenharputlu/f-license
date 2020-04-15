package main

import (
	"crypto/tls"
	"f-license/config"
	"fmt"

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

func main() {

	intro()

	r := mux.NewRouter()
	// Endpoints called by product owners
	adminRouter := r.PathPrefix("/admin").Subrouter()
	adminRouter.Use(AuthenticationMiddleware)
	adminRouter.HandleFunc("/generate", GenerateLicense).Methods(http.MethodPost)
	adminRouter.HandleFunc("/{id}", GetLicense).Methods(http.MethodGet)
	adminRouter.HandleFunc("/delete/{id}", DeleteLicense).Methods(http.MethodDelete)
	adminRouter.HandleFunc("/activate/{id}", ChangeLicenseActiveness).Methods(http.MethodPut)
	adminRouter.HandleFunc("/inactivate/{id}", ChangeLicenseActiveness).Methods(http.MethodPut)

	// Endpoints called by product instances having license
	r.HandleFunc("/license/verify", VerifyLicense).Methods(http.MethodPost)
	r.HandleFunc("/license/ping", Ping).Methods(http.MethodPost)

	addr := fmt.Sprintf(":%d", config.Global.Port)
	certFile := config.Global.ServerOptions.CertFile
	keyFile := config.Global.ServerOptions.KeyFile

	if config.Global.ServerOptions.EnableTLS {
		srv := &http.Server{
			Addr:         addr,
			Handler:      r,
			TLSConfig:    &config.Global.ServerOptions.TLSConfig,
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
		}
		log.Fatal(srv.ListenAndServeTLS(certFile, keyFile))
	} else {
		log.Fatal(http.ListenAndServe(addr, r))
	}
}
