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
	ControlAPISecret string          `json:"control_api_secret"`
	Secret           string          `json:"secret"`
	Apps             map[string]*App `json:"apps"`
	DefaultKey       Key             `json:"default_key"`
	ServerOptions    ServerOptions   `json:"server_options"`
	DBOptions        DBOptions       `json:"db_options"`
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

type Key struct {
	ID   string     `bson:"id" json:"id"`
	Name string     `bson:"name" json:"name"`
	Type string     `bson:"type" json:"type"`
	RSA  *RSA       `bson:"rsa,omitempty" json:"rsa,omitempty"`
	HMAC *KeyDetail `bson:"hmac,omitempty" json:"hmac,omitempty"`
}

type RSA struct {
	Private *KeyDetail `bson:"private,omitempty" json:"private,omitempty"`
	Public  *KeyDetail `bson:"public,omitempty" json:"public,omitempty"`
}

type KeyDetail struct {
	FilePath  string `bson:"-" json:"file_path,omitempty"`
	Raw       string `bson:"-" json:"raw"`
	Encrypted []byte `bson:"encrypted,omitempty" json:"-"`
}

type ServerOptions struct {
	EnableTLS bool       `json:"enable_tls"`
	CertFile  string     `json:"cert_file"`
	KeyFile   string     `json:"key_file"`
	TLSConfig tls.Config `json:"tls_config"`
}

type App struct {
	Name string `json:"name"`
	Alg  string `json:"alg"`
	Key  Key    `json:"key"`
}

type DBOptions struct {
	Type  string `json:"type"`
	File  File   `json:"file"`
	Mongo Mongo  `json:"mongo"`
}

type File struct {
	Dir string `json:"dir"`
}

type Mongo struct {
	URL    string `json:"url"`
	Name string `json:"name"`
}
