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

}

func fatalf(format string, err error) {
	if err != nil {
		logrus.Fatalf(format, err)
	}
}

type License struct {
	ID        primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Headers   map[string]interface{} `bson:"headers" json:"headers"`
	Hash      string                 `bson:"hash" json:"-"`
	Token     string                 `bson:"token" json:"token"`
	Claims    jwt.MapClaims          `bson:"claims" json:"claims"`
	Active    bool                   `bson:"active" json:"active"`
	signKey   interface{}
	verifyKey interface{}
}

func (l *License) GetAppName() (appName string) {
	app, ok := l.Headers["app"]
	if ok {
		appName = app.(string)
		return
	}

	return
}

// GetAlg returns alg defined in the license header.
func (l *License) GetAlg() (alg string) {
	app, ok := l.Headers["alg"]
	if ok {
		alg = app.(string)
		return
	}

	return
}

// GetApp returns the app that will be used to generate a license. Firstly, it checks whether a special app defined
// in config. If there is no app defined with the given name, it picks configured DefaultApp as a fallback. Then,
// if an alg is defined in license level, it overrides app alg, if not uses the app's existing alg. If there is no alg
// defined, picks HS256 as default.
func (l *License) GetApp(appName string) *config.App {
	var alg string

	app, ok := config.Global.Apps[appName]
	if !ok {
		logrus.Warn("There is no valid app found, using default app")
		app = config.Global.DefaultApp
	}

	if app == nil {
		logrus.Fatal("Default app is also not found, please define a default app")
	}

	alg = app.Alg

	if licenseLevelAlg := l.GetAlg(); licenseLevelAlg != "" {
		logrus.Warn("Overriding alg with the one defined in license header")
		alg = licenseLevelAlg
	}

	if alg == "" {
		logrus.Warn("No alg defined: choosing HS256")
		alg = "HS256"
	}

	l.Headers["alg"] = alg

	return app
}

func (l *License) Generate() error {

	if len(l.Headers) == 0 {
		l.Headers = make(map[string]interface{})
	}

	app := l.GetApp(l.GetAppName())

	token := jwt.NewWithClaims(jwt.GetSigningMethod(app.Alg), l.Claims)
	token.Header = l.Headers

	l.LoadSignKey(app)
	l.LoadVerifyKey(app)

	signedString, err := token.SignedString(l.signKey)
	if err != nil {
		return err
	}

	l.Token = signedString

	h := fnv.New64a()
	h.Write([]byte(signedString))
	l.Hash = fmt.Sprintf("%v", h.Sum64())

	return nil
}

func (l *License) LoadSignKey(app *config.App) {

	if strings.HasPrefix(app.Alg, "HS") {
		l.signKey = []byte(app.HMACSecret)
	} else {
		signBytes, err := ioutil.ReadFile(app.RSAPrivateKeyFile)
		fatalf("Couldn't read rsa private key file: %s", err)

		l.signKey, err = jwt.ParseRSAPrivateKeyFromPEM(signBytes)
		fatalf("Couldn't parse private key: %s", err)
	}
}

func (l *License) LoadVerifyKey(app *config.App) {

	if strings.HasPrefix(app.Alg, "HS") {

		l.verifyKey = []byte(app.HMACSecret)
	} else {
		verifyBytes, err := ioutil.ReadFile(app.RSAPublicKeyFile)
		fatalf("Couldn't read public key: %s", err)

		l.verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
		fatalf("Couldn't parse public key: %s", err)
	}
}

func (l *License) IsLicenseValid(tokenString string) (bool, error) {
	if !l.Active {
		return false, nil
	}

	if l.verifyKey == nil {
		app := l.GetApp(l.GetAppName())
		l.LoadVerifyKey(app)
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		switch token.Method.(type) {
		case *jwt.SigningMethodHMAC:
			return l.verifyKey, nil
		case *jwt.SigningMethodRSA:

			return l.verifyKey, nil
		default:
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
	})

	return token.Valid, err
}
