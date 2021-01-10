package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"hash/fnv"
	"io/ioutil"
	"os"

	"github.com/furkansenharputlu/f-license/lcs"
	"github.com/furkansenharputlu/f-license/storage/storage"
	"github.com/furkansenharputlu/f-license/config"

	"github.com/sirupsen/logrus"
)

var store []* lcs.License

var file = config.Global.DatabaseOptions.Default.Path+config.Global.DatabaseOptions.Default.FileName

func Connect(){
	Read(file)
}

func AddIfNotExisting(l* lcs.License)  error{
	for _, tmp := range store {
		if tmp.Hash == l.Hash{
			return errors.New(fmt.Sprintf("there is already such license with ID: %s", tmp.ID/*.Hex()*/))		}
	}
	store = append(store, l)
	Write(file)
	return nil
}

func Activate(id string, inactivate bool) error {

	licenseID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New(fmt.Sprintf("ID format error: %s", err))
	}
	for _, tmp := range store {
		if tmp.ID == licenseID {
			if inactivate == true {
				if tmp.Active == false {
					return errors.New("already inactive")
				} else {
					tmp.Active = false
					Write(file)
					logrus.Infof(`License is successfully inactivated: %s`, id)
					return nil
				}
			} else {
				if tmp.Active == false {
					tmp.Active = true
					Write(file)
					logrus.Infof(`License is successfully activated: %s`, id)
					return nil
				} else {
					return errors.New("already active")
				}
			}
		}
	}
	return errors.New("there is no matching license")
}

func DeleteByID(id string) error {

		licenseID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return errors.New(fmt.Sprintf("ID format error: %s", err))
		}
	for i, tmp := range store{
		if tmp.ID == licenseID{
			store = append(store[:i], store[i+1:]...)
			Write(file)
			logrus.Info("License successfully deleted")
			return nil
		}
	}
	return errors.New(fmt.Sprintf("there is no license with ID: %s", licenseID))
}

func GetByID(id string, l *lcs.License) error {

		licenseID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return errors.New(fmt.Sprintf("ID format error: %s", err))
		}

	for _, tmp := range store{
		if tmp.ID == licenseID{
			*l = *tmp
			return nil
		}
	}
	return errors.New(fmt.Sprintf("there is no license with ID: %s", licenseID))
}


func GetAll(licenses *[]*lcs.License) error {
	*licenses= store
	return nil
}
func GetByToken(token string, l *lcs.License) error {
	h64 := fnv.New64a()
	h64.Write([]byte(token))
	hash := h64.Sum64()
	hashStr := fmt.Sprintf("%v", hash)

	for _, tmp := range store{
		if tmp.Token == hashStr{
			*l = *tmp
			return nil
		}
	}
	return errors.New(fmt.Sprintf("there is no license with token: %s", token))

}
func DropDatabase() error {

	err := os.Remove("./test.json")

	if err != nil {
		fmt.Println(err)
		return errors.New("The database could not dropped")
	}
	store=nil
	Write(file)

	return nil
}

func Write(path string) error{
	bytes, err := json.Marshal(store)
	if err !=nil{
		return errors.New("Couldn't marshal configuration")
	}
	err = ioutil.WriteFile(path, bytes, 0644)
	if err != nil {
		return errors.New("Couldn't write to data file")
	}
	return nil
}

func Read(path string) error{
	configuration, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.New("Couldn't read to data file")
	}
	err = json.Unmarshal(configuration, &store)
	if err != nil {
		return errors.New("Couldn't unmarshal configuration")
	}
	return nil
}
