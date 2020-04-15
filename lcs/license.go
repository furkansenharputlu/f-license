package lcs

import (
	"context"
	"errors"
	"f-license/config"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"hash/fnv"
	"time"
)

var licensesCol *mongo.Collection

func init() {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	MongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(config.Global.MongoURL))
	if err != nil {
		logrus.Fatalf("Problem while connecting to Mongo: %s", err)
	}

	licensesCol = MongoClient.Database("f-license").Collection("licenses")
}

type License struct {
	ID     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Type   string             `bson:"type" json:"type"`
	Hash   string             `bson:"hash" json:"-"`
	Token  string             `bson:"token" json:"token"`
	Claims jwt.MapClaims      `bson:"claims" json:"claims"`
	Active bool               `bson:"active" json:"active"`
}

func (l *License) Add() error {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, l.Claims)
	signedString, err := token.SignedString([]byte(config.Global.Secret))
	if err != nil {
		logrus.Error("Error signing token:", err)
	}

	l.Token = signedString

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	h := fnv.New64a()
	h.Write([]byte(signedString))
	l.Hash = fmt.Sprintf("%v", h.Sum64())

	filter := bson.M{"hash": l.Hash}
	res := licensesCol.FindOne(ctx, filter)
	err = res.Err()
	if err != nil {
		if err != mongo.ErrNoDocuments {
			return err
		}
	} else {
		var existingLicense License
		_ = res.Decode(&existingLicense)
		return errors.New(fmt.Sprintf("there is already such license with ID: %s", existingLicense.ID.Hex()))
	}

	l.ID = primitive.NewObjectID()

	update := bson.M{"$set": l}
	_, err = licensesCol.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return errors.New(fmt.Sprintf("error while inserting license: %s", err))
	}

	logrus.Info("License successfully generated")

	return nil
}

func (l *License) GetByID(id string) error {
	licenseID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New(fmt.Sprintf("ID format error: %s", err))
	}

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	filter := bson.M{"_id": licenseID}
	res := licensesCol.FindOne(ctx, filter)
	err = res.Err()
	if err != nil {
		return err
	}

	_ = res.Decode(l)

	return nil
}

func (l *License) GetByToken(license string) error {
	h := fnv.New64a()
	h.Write([]byte(license))
	hash := h.Sum64()
	hashStr := fmt.Sprintf("%v", hash)

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	filter := bson.M{"hash": hashStr}
	res := licensesCol.FindOne(ctx, filter)
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

func (l *License) Activate(id string, inactivate bool) error {
	licenseID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New(fmt.Sprintf("ID format error: %s", err))
	}

	filter := bson.M{"_id": bson.M{"$eq": licenseID}}
	update := bson.M{"$set": bson.M{"active": !inactivate}}
	res, err := licensesCol.UpdateOne(context.Background(), filter, update)
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

func (l *License) IsLicenseValid(license string) (bool, error) {

	if !l.Active {
		return false, fmt.Errorf("license inactivated")
	}

	token, err := jwt.Parse(license, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(config.Global.Secret), nil
	})

	if err != nil {
		logrus.Error(err)
		return false, err
	}

	if !token.Valid {
		return false, nil
	}

	return true, nil
}

func (l *License) DeleteByID(id string) error {
	licenseID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New(fmt.Sprintf("ID format error: %s", err))
	}

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	filter := bson.M{"_id": licenseID}
	res, err := licensesCol.DeleteOne(ctx, filter)
	if res.DeletedCount == 0 {
		return errors.New(fmt.Sprintf("there is no license with ID: %s", id))
	}

	if err != nil {
		return errors.New("license cannot be deleted")
	}

	logrus.Info("License successfully deleted")

	return nil
}
