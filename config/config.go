package config

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io/ioutil"
)

var Global = &Config{}

type Config struct {
	Secret string
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
