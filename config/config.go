package config

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io/ioutil"
)

var Global = &Config{}

type Config struct {
	Secret      string `json:"secret"`
	Port        int    `json:"port"`
	AdminSecret string `json:"admin_secret"`
	MongoURL    string `json:"mongo_url"`
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
