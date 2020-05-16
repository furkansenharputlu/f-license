package config

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"

	"github.com/sirupsen/logrus"
)

var Global = &Config{}

type Config struct {
	Port             int             `json:"port"`
	AdminSecret      string          `json:"admin_secret"`
	Apps             map[string]*App `json:"apps"`
	DefaultSignature Signature       `json:"default_signature"`
	MongoURL         string          `json:"mongo_url"`
	DBName           string          `json:"db_name"`
	ServerOptions    ServerOptions   `json:"server_options"`
}

type Signature struct {
	HMACSecret        string `json:"hmac_secret"`
	RSAPrivateKeyFile string `json:"rsa_private_key_file"`
	RSAPublicKeyFile  string `json:"rsa_public_key_file"`
}

func (c *Config) Load(filePath string) {
	configuration, err := ioutil.ReadFile(filePath)
	if err != nil {
		logrus.WithError(err).Error("Couldn't read config file")
	}

	err = json.Unmarshal(configuration, &c)
	if err != nil {
		logrus.WithError(err).Error("Couldn't unmarshal configuration")
	}
}

type ServerOptions struct {
	EnableTLS bool       `json:"enable_tls"`
	CertFile  string     `json:"cert_file"`
	KeyFile   string     `json:"key_file"`
	TLSConfig tls.Config `json:"tls_config"`
}

type App struct {
	Name      string    `json:"name"`
	Alg       string    `json:"alg"`
	Signature Signature `json:"signature"`
}
