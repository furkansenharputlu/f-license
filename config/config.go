package config

import (
	"crypto/tls"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io/ioutil"
)

var Global = &Config{}

type Config struct {
	Secret        string        `json:"secret"`
	Port          int           `json:"port"`
	AdminSecret   string        `json:"admin_secret"`
	MongoURL      string        `json:"mongo_url"`
	ServerOptions ServerOptions `json:"server_options"`
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

func init(){
	Global.Load("config.json")
}
