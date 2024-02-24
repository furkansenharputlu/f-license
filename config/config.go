package config

import (
	"crypto/tls"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/furkansenharputlu/f-license/storage"
	"io/ioutil"
	"time"

	"github.com/sirupsen/logrus"
)

var Global = &Config{}

type Product struct {
	ID        string    `json:"id"`
	Name      string    `json:"name" gorm:"uniqueIndex"`
	Alg       string    `json:"alg"`
	KeyID     string    `json:"key"`
	Key       *Key      `json:"-"`
	Plans     Plans     `json:"plans"`
	CreatedAt time.Time `json:"created_at"`
}

type GORMMap map[string]interface{}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (m *GORMMap) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	var result map[string]interface{}
	err := json.Unmarshal(bytes, &result)
	*m = result
	return err
}

// Value return json value, implement driver.Valuer interface
func (m GORMMap) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}
	return json.Marshal(m)
}

type Plans []*Plan

// Scan scan value into Jsonb, implements sql.Scanner interface
func (p *Plans) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	var result []*Plan
	err := json.Unmarshal(bytes, &result)
	*p = result
	return err
}

// Value return json value, implement driver.Valuer interface
func (p Plans) Value() (driver.Value, error) {
	if len(p) == 0 {
		return nil, nil
	}
	return json.Marshal(p)
}

func (Plans) GormDataType() string {
	return "plans"
}

type Policy struct {
	Expiration int  `json:"expiration"`
	Headers    []KV `json:"headers"`
	Claims     []KV `json:"claims"`
}

type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Plan struct {
	Name        string   `json:"name"`
	Price       float64  `json:"price"`
	Currency    string   `json:"currency"`
	Features    []string `json:"features"`
	ButtonLabel string   `json:"buttonLabel"`
	Policy      Policy   `json:"policy"`
}

func (p *Product) GetKey() (*Key, error) {
	var key Key
	err := storage.SQLHandler.Get(&key, "id = ?", p.KeyID)
	if err != nil {
		return nil, err
	}

	return &key, nil
}

type Config struct {
	Port               int                `json:"port"`
	ControlAPISecret   string             `json:"control_api_secret"`
	Secret             string             `json:"secret"`
	LoadProductsFromDB bool               `json:"load_products_from_db"`
	Products           map[string]Product `json:"products"`
	DefaultKey         Key                `json:"default_key"`
	MongoURL           string             `json:"mongo_url"`
	DBName             string             `json:"db_name"`
	ServerOptions      ServerOptions      `json:"server_options"`
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
	ID          string    `json:"id"`
	Name        string    `json:"name" gorm:"uniqueIndex"` // name should be unique
	Type        string    `json:"type"`
	HMAC        string    `json:"hmac,omitempty"`
	HMACPath    string    `json:"hmac_path,omitempty" gorm:"-"`
	Private     string    `json:"private,omitempty"`
	PrivatePath string    `json:"private_path,omitempty" gorm:"-"`
	Public      string    `json:"public,omitempty"`
	PublicPath  string    `json:"public_path,omitempty" gorm:"-"`
	CreatedAt   time.Time `json:"created_at"`
}

type KeyInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func (KeyInfo) TableName() string {
	return "keys"
}

type ServerOptions struct {
	EnableTLS bool       `json:"enable_tls"`
	CertFile  string     `json:"cert_file"`
	KeyFile   string     `json:"key_file"`
	TLSConfig tls.Config `json:"tls_config"`
}
