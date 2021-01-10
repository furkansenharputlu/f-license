package storage

import (

	"github.com/furkansenharputlu/f-license/lcs"
	"github.com/sirupsen/logrus"
)

type Handler interface {
	Connect()
	AddIfNotExisting(l *lcs.License) error
	Activate(id string, inactivate bool) error
	GetByID(id string, l *lcs.License) error
	GetAll(licenses *[]*lcs.License) error
	GetByToken(token string, l *lcs.License) error
	DeleteByID(id string) error
	DropDatabase() error
}

var LicenseHandler Handler

func fatalf(format string, err error) {
	if err != nil {
		logrus.Fatalf(format, err)
	}
}