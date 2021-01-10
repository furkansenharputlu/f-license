package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/furkansenharputlu/f-license/lcs"
)

const licensesDir = "licenses"

type FileHandler struct {
}

func (f FileHandler) AddIfNotExisting(l *lcs.License) (error, int) {
	if _, err := os.Stat(withJSONExtension(l.ID)); !os.IsNotExist(err) {
		return errors.New(fmt.Sprintf("there is already such license with ID: %s", l.ID)), ItemDuplicationError
	}

	return f.Write(l), NoError
}

func (f FileHandler) Activate(id string, inactivate bool) error {
	/*
		licenseID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return errors.New(fmt.Sprintf("ID format error: %s", err))
		}
		for _, tmp := range f.store {
			if tmp.ID == licenseID {
				if inactivate == true {
					if tmp.Active == false {
						return errors.New("already inactive")
					} else {
						tmp.Active = false
						f.Write(f.file)
						logrus.Infof(`License is successfully inactivated: %s`, id)
						return nil
					}
				} else {
					if tmp.Active == false {
						tmp.Active = true
						f.Write(f.file)
						logrus.Infof(`License is successfully activated: %s`, id)
						return nil
					} else {
						return errors.New("already active")
					}
				}
			}
		}

		return errors.New("there is no matching license")*/
	return nil
}

func (f FileHandler) DeleteByID(id string) error {
	return os.Remove(withJSONExtension(id))
}

func (f FileHandler) GetByID(id string, l *lcs.License) error {
	licenseInBytes, err := ioutil.ReadFile(withJSONExtension(id))
	if err != nil {
		return errors.New(fmt.Sprintf("couldn't read to license with ID: %s", id))
	}

	err = json.Unmarshal(licenseInBytes, &l)
	if err != nil {
		return errors.New(fmt.Sprintf("couldn't unmarshal license with ID: %s", id))
	}

	return nil
}

func (f FileHandler) GetAll(licenses *[]*lcs.License) error {
	licenseFiles, err := ioutil.ReadDir(licensesDir)
	if err != nil {
		return errors.New("couldn't read licenses")
	}

	for _, licenseFile := range licenseFiles {
		var l lcs.License
		if err := f.GetByID(removeExtension(licenseFile.Name()), &l); err != nil {
			return err
		}
		*licenses = append(*licenses, &l)
	}

	return nil
}

func (f FileHandler) GetByToken(token string, l *lcs.License) error {
	/*h64 := fnv.New64a()
	h64.Write([]byte(token))
	hash := h64.Sum64()
	hashStr := fmt.Sprintf("%v", hash)
	*/
	return nil
}

func (f FileHandler) DropDatabase() error {
	return os.RemoveAll(licensesDir)
}

func (f FileHandler) Write(l *lcs.License) error {
	bytes, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return errors.New("couldn't marshal configuration")
	}

	if _, err := os.Stat(licensesDir); os.IsNotExist(err) {
		_ = os.Mkdir(licensesDir, 0777)
	}

	err = ioutil.WriteFile(withJSONExtension(l.ID), bytes, 0644)
	if err != nil {
		return errors.New(fmt.Sprintf("couldn't write license to file: %s", err))
	}

	return nil
}

func withJSONExtension(filename string) string {
	return filepath.Join(licensesDir, filename+".json")
}

func removeExtension(filename string) string {
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}
