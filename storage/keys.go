package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/furkansenharputlu/f-license/config"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type KeyHandler interface {
	AddIfNotExisting(k *config.Key) error
	//Activate(id string, inactivate bool) error
	GetByID(id string, k *config.Key) error
	GetAll(keys *[]*config.Key) error
	//GetByToken(token string, l *lcs.License) error
	DeleteByID(id string) error
	//DropDatabase() error
}

var GlobalKeyHandler KeyHandler

type mongoKeyHandler struct {
	col *mongo.Collection
}

func (h mongoKeyHandler) AddIfNotExisting(k *config.Key) error {

	filter := bson.M{"id": k.ID}
	res := h.col.FindOne(context.Background(), filter)
	err := res.Err()
	if err != nil {
		if err != mongo.ErrNoDocuments {
			return err
		}
	} else {
		var existingKey config.Key
		_ = res.Decode(&existingKey)
		return errors.New(fmt.Sprintf("there is already such key with ID: %s", existingKey.ID))
	}

	update := bson.M{"$set": k}
	_, err = h.col.UpdateOne(context.Background(), filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return errors.New(fmt.Sprintf("error while inserting key: %s", err))
	}

	return nil
}

func (h mongoKeyHandler) GetByID(keyID string, s *config.Key) error {

	filter := bson.M{"id": keyID}
	res := h.col.FindOne(context.Background(), filter)
	err := res.Err()
	if err != nil {
		return err
	}

	_ = res.Decode(s)

	return nil
}

func (h mongoKeyHandler) DeleteByID(id string) error {
	filter := bson.M{"id": id}
	res, err := h.col.DeleteOne(context.Background(), filter)
	if res.DeletedCount == 0 {
		return errors.New(fmt.Sprintf("there is no key with ID: %s", id))
	}

	if err != nil {
		return errors.New("key cannot be deleted")
	}

	logrus.Info("Key successfully deleted")

	return nil
}

func (h mongoKeyHandler) GetAll(keys *[]*config.Key) error {
	cur, err := h.col.Find(context.Background(), bson.D{})
	if err != nil {
		return err
	}

	defer cur.Close(context.Background())

	for cur.Next(context.Background()) {

		var k config.Key
		err := cur.Decode(&k)
		if err != nil {
			return err
		}

		*keys = append(*keys, &k)

	}

	return cur.Err()
}

/*type RSAHandler interface {
	AddIfNotExisting(rsa *config.RSA) error
	//Activate(id string, inactivate bool) error
	GetByID(id string, rsa *config.RSA) error
	//GetAll(licenses *[]*lcs.License) error
	//GetByToken(token string, l *lcs.License) error
	//DeleteByID(id string) error
	//DropDatabase() error
}

var GlobalRSAHandler RSAHandler

type mongoRSAHandler struct {
	col *mongo.Collection
}

func (h mongoRSAHandler) AddIfNotExisting(rsa *config.RSA) error {

	filter := bson.M{"name": rsa.Name}
	res := h.col.FindOne(context.Background(), filter)
	err := res.Err()
	if err != nil {
		if err != mongo.ErrNoDocuments {
			return err
		}
	} else {
		var existingRSA config.RSA
		_ = res.Decode(&existingRSA)
		return errors.New(fmt.Sprintf("there is already such rsa pair with name: %s", existingRSA.Name))
	}

	update := bson.M{"$set": rsa}
	_, err = h.col.UpdateOne(context.Background(), filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return errors.New(fmt.Sprintf("error while inserting rsa pair: %s", err))
	}

	return nil
}

func (h mongoRSAHandler) GetByID(id string, rsa *config.RSA) error {

	filter := bson.M{"id": id}
	res := h.col.FindOne(context.Background(), filter)
	err := res.Err()
	if err != nil {
		return err
	}

	_ = res.Decode(rsa)

	return nil
}*/
