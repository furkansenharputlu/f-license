package storage

import (
	"gorm.io/driver/sqlite"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Handler interface {
	AddIfNotExisting(item interface{}) error
	Activate(id string, inactivate bool) error
	Get(item interface{}, query interface{}, args ...interface{}) error
	GetAll(items interface{}) error
	Update(model interface{}, values interface{}) error
	Delete(item interface{}, query interface{}, args ...interface{}) error
	DropDatabase() error
	DB() interface{}
}

type sqlHandler struct {
	db *gorm.DB
}

func (s sqlHandler) DB() interface{} {
	return s.db
}

func (s sqlHandler) AddIfNotExisting(item interface{}) error {
	return s.db.Create(item).Error
}

func (sqlHandler) Activate(id string, inactivate bool) error {
	panic("implement me")
}

func (s sqlHandler) Get(item interface{}, query interface{}, args ...interface{}) error {
	return s.db.Where(query, args...).Find(item).Error
}

func (s sqlHandler) GetAll(items interface{}) error {
	return s.db.Order("created_at DESC").Find(items).Error
}

func (s sqlHandler) Update(model interface{}, values interface{}) error {
	return s.db.Model(model).Updates(values).Error
}

func (s sqlHandler) Delete(item interface{}, query interface{}, args ...interface{}) error {
	return s.db.Where(query, args...).Delete(item).Error
}

func (sqlHandler) DropDatabase() error {
	panic("implement me")
}

var SQLHandler Handler

func Connect(dst ...interface{}) {
	db, err := gorm.Open(sqlite.Open("f-license.db"), &gorm.Config{})
	if err != nil {
		fatalf("Problem while connecting to SQL: %s", err)
	}

	err = db.AutoMigrate(dst...)
	if err != nil {
		fatalf("Problem while creating SQL tables: %s", err)
	}

	SQLHandler = sqlHandler{db: db}
}

func fatalf(format string, err error) {
	if err != nil {
		logrus.Fatalf(format, err)
	}
}
