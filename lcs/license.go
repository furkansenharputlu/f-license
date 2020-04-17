package lcs

import (
	"f-license/config"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var VerifyKey interface{}

func ReadKeys() {

	if config.Global.ServerOptions.CertFile != "" {
		verifyBytes, err := ioutil.ReadFile(config.Global.ServerOptions.CertFile)
		fatalf("Couldn't read public key: %s", err)

		VerifyKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
		fatalf("Couldn't parse public key: %s", err)
	}
}

func fatalf(format string, err error) {
	if err != nil {
		logrus.Fatalf(format, err)
	}
}

type License struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Type    string             `bson:"type" json:"type"`
	Alg     string             `bson:"alg" json:"alg"`
	Hash    string             `bson:"hash" json:"-"`
	Token   string             `bson:"token" json:"token"`
	Claims  jwt.MapClaims      `bson:"claims" json:"claims"`
	Active  bool               `bson:"active" json:"active"`
	signKey interface{}
}

func (l *License) Generate() error {
	if l.Alg == "" {
		l.Alg = "HS256"
	}
	token := jwt.NewWithClaims(jwt.GetSigningMethod(l.Alg), l.Claims)

	l.LoadSignKey()

	signedString, err := token.SignedString(l.signKey)
	if err != nil {
		logrus.Errorf("Error while signing token: %s", err)
	}

	l.Token = signedString

	h := fnv.New64a()
	h.Write([]byte(signedString))
	l.Hash = fmt.Sprintf("%v", h.Sum64())

	return nil
}

func (l *License) LoadSignKey() {
	if strings.HasPrefix(l.Alg, "HS") {
		l.signKey = []byte(config.Global.HMACSecret)
	} else {
		signBytes, err := ioutil.ReadFile(config.Global.RSAPrivateKeyFile)
		fatalf("Couldn't read rsa private key file: %s", err)

		l.signKey, err = jwt.ParseRSAPrivateKeyFromPEM(signBytes)
		fatalf("Couldn't parse private key: %s", err)
	}
}

func (l *License) IsLicenseValid(tokenString string) (bool, error) {
	if !l.Active {
		return false, nil
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		switch token.Method.(type) {
		case *jwt.SigningMethodHMAC:
			return []byte(config.Global.HMACSecret), nil
		case *jwt.SigningMethodRSA:
			return VerifyKey, nil
		default:
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
	})

	return token.Valid, err
}
