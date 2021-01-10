package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/furkansenharputlu/f-license/lcs"
	"hash/fnv"
	"time"

	"github.com/furkansenharputlu/f-license/config"
	"github.com/furkansenharputlu/f-license/storage/storage"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/sirupsen/logrus"

)

type licenseMongoHandler struct {
	col *mongo.Collection
}

func (h licenseMongoHandler) Connect() {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	Mongo:= config.Global.DatabaseOptions.Mongo
	URI := fmt.Sprintf("%s://%s:%d", Mongo.Type,Mongo.Host,Mongo.Port)

	opt:=options.Client().ApplyURI(URI)

	if Mongo.Auth {
		credential:=options.Credential{
			Username: Mongo.Username,
			Password: Mongo.Password,
		}
		opt.SetAuth(credential)
	}

	MongoClient, err := mongo.Connect(ctx, opt)
	fatalf("Problem while connecting to Mongo: %s", err)
	LicenseHandler = licenseMongoHandler{MongoClient.Database(Mongo.DBName).Collection("licenses")}
}
func (h licenseMongoHandler) AddIfNotExisting(l *lcs.License) error {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	filter := bson.M{"hash": l.Hash}
	res := h.col.FindOne(ctx, filter)
	err := res.Err()
	if err != nil {
		if err != mongo.ErrNoDocuments {
			return err
		}
	} else {
		var existingLicense lcs.License
		_ = res.Decode(&existingLicense)
		return errors.New(fmt.Sprintf("there is already such license with ID: %s", existingLicense.ID.Hex()))
	}

	l.ID = primitive.NewObjectID()

	update := bson.M{"$set": l}
	_, err = h.col.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return errors.New(fmt.Sprintf("error while inserting license: %s", err))
	}

	return nil
}

func (h licenseMongoHandler) Activate(id string, inactivate bool) error {
	licenseID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New(fmt.Sprintf("ID format error: %s", err))
	}

	filter := bson.M{"_id": bson.M{"$eq": licenseID}}
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
	licenseID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New(fmt.Sprintf("ID format error: %s", err))
	}

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	filter := bson.M{"_id": licenseID}
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
	licenseID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New(fmt.Sprintf("ID format error: %s", err))
	}

	filter := bson.M{"_id": licenseID}
	res := h.col.FindOne(context.Background(), filter)
	err = res.Err()
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

