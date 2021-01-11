package storage

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/furkansenharputlu/f-license/config"
	"github.com/furkansenharputlu/f-license/lcs"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	NoError                = 0
	UnexpectedFailureError = 1
	ItemDuplicationError   = 2
)

type Handler interface {
	AddIfNotExisting(l *lcs.License) (error, int)
	Activate(id string, inactivate bool) error
	GetByID(id string, l *lcs.License) error
	GetAll(licenses *[]*lcs.License) error
	GetByToken(token string, l *lcs.License) error
	DeleteByID(id string) error
	DropDatabase() error
}

var LicenseHandler Handler

func Connect() {
	if config.Global.DBOptions.Type == "mongo" {
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		MongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(config.Global.DBOptions.Mongo.URL))
		fatalf("Problem while connecting to Mongo: %s", err)

		LicenseHandler = licenseMongoHandler{MongoClient.Database(config.Global.DBOptions.Mongo.Name).Collection("licenses")}
		GlobalKeyHandler = mongoKeyHandler{MongoClient.Database(config.Global.DBOptions.Mongo.Name).Collection("keys")}
		//GlobalRSAHandler = mongoRSAHandler{MongoClient.Database(config.Global.DBName).Collection("keys")}
	} else {
		LicenseHandler = FileHandler{}
	}
}

func fatalf(format string, err error) {
	if err != nil {
		logrus.Fatalf(format, err)
	}
}

type licenseMongoHandler struct {
	col *mongo.Collection
}

func (h licenseMongoHandler) AddIfNotExisting(l *lcs.License) (error, int) {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	filter := bson.M{"id": l.ID}
	res := h.col.FindOne(ctx, filter)
	err := res.Err()
	if err != nil {
		if err != mongo.ErrNoDocuments {
			return err, UnexpectedFailureError
		}
	} else {
		var existingLicense lcs.License
		_ = res.Decode(&existingLicense)
		return errors.New(fmt.Sprintf("there is already such license with ID: %s", existingLicense.ID)), ItemDuplicationError
	}

	l.ID = lcs.HexSHA256([]byte(l.Token))

	update := bson.M{"$set": l}
	_, err = h.col.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return errors.New(fmt.Sprintf("error while inserting license: %s", err)), 0
	}

	return nil, NoError
}

func (h licenseMongoHandler) Activate(id string, inactivate bool) error {
	filter := bson.M{"id": bson.M{"$eq": id}}
	update := bson.M{"$set": bson.M{"active": !inactivate}}
	res, err := h.col.UpdateOne(context.Background(), filter, update)
	if res.MatchedCount == 0 {
		return errors.New("there is no matching license")
	}

	if res.ModifiedCount == 0 {
		if inactivate {
			return errors.New("already inactive")
		} else {
			return errors.New("already active")
		}
	}

	if err != nil {
		return errors.New("license cannot be updated")
	}

	if inactivate {
		logrus.Infof(`License is successfully inactivated: %s`, id)
	} else {
		logrus.Infof(`License is successfully activated: %s`, id)
	}

	return nil
}

func (h licenseMongoHandler) DeleteByID(id string) error {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	filter := bson.M{"id": id}
	res, err := h.col.DeleteOne(ctx, filter)
	if res.DeletedCount == 0 {
		return errors.New(fmt.Sprintf("there is no license with ID: %s", id))
	}

	if err != nil {
		return errors.New("license cannot be deleted")
	}

	logrus.Info("License successfully deleted")

	return nil
}

func (h licenseMongoHandler) GetByID(id string, l *lcs.License) error {
	filter := bson.M{"id": id}
	res := h.col.FindOne(context.Background(), filter)
	err := res.Err()
	if err != nil {
		return err
	}

	_ = res.Decode(l)

	return nil
}

func (h licenseMongoHandler) GetAll(licenses *[]*lcs.License) error {
	cur, err := h.col.Find(context.Background(), bson.D{})
	if err != nil {
		return err
	}

	defer cur.Close(context.Background())

	for cur.Next(context.Background()) {

		var l lcs.License
		err := cur.Decode(&l)
		if err != nil {
			return err
		}

		*licenses = append(*licenses, &l)

	}

	return cur.Err()
}

func (h licenseMongoHandler) GetByToken(token string, l *lcs.License) error {
	// TODO: Refactor hashing
	h64 := fnv.New64a()
	h64.Write([]byte(token))
	hash := h64.Sum64()
	hashStr := fmt.Sprintf("%v", hash)

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	filter := bson.M{"hash": hashStr}
	res := h.col.FindOne(ctx, filter)
	err := res.Err()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("license not found")
		}
		return fmt.Errorf("error while getting license: %s", err)
	}

	_ = res.Decode(l)

	return nil
}

func (h licenseMongoHandler) DropDatabase() error {
	return h.col.Database().Drop(context.Background())
}
