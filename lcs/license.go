package lcs

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"

	"github.com/dgrijalva/jwt-go"
	"github.com/furkansenharputlu/f-license/config"
)

type License struct {
	ID      string                 `bson:"id" json:"id"`
	Active  bool                   `bson:"active" json:"active"`
	Headers map[string]interface{} `bson:"headers" json:"headers"`
	Claims  jwt.MapClaims          `bson:"claims" json:"claims"`
	Token   string                 `bson:"token" json:"token"`
	Key     config.Key             `bson:"key" json:"key"`

	SignKey   interface{} `bson:"-" json:"-"`
	VerifyKey interface{} `bson:"-" json:"-"`
}

/*func (l *License) SigningMethod() string{

}

func (l *License) AlgKeyTypeMatches() bool {
	if stringsl.GetAlg()

	return
}*/

/*func (l *License) MarshalJSON() ([]byte, error) {

	res := map[string]interface{}{
		"id":      l.ID,
		"headers": l.Headers,
		"token":   l.Token,
		"claims":  l.Claims,
		"active":  l.Active,
	}

	return json.Marshal(&struct {
		ID           string                 `json:"id"`
		Headers      map[string]interface{} `json:"headers"`
		Token        string                 `json:"token"`
		Claims       jwt.MapClaims          `json:"claims"`
		Active       bool                   `json:"active"`
		HMACSecretID string                 `json:"hmac_secret_id,omitempty"`
		RSAID        string                 `json:"rsa_id,omitempty"`
	}{
		ID:      l.ID,
		Headers: l.Headers,
		Token:   l.Token,
		Claims:  l.Claims,
		Active:  l.Active,
	})



	return json.Marshal(res)
}*/

func (l *License) GetAppName() (appName string) {
	app, ok := l.Headers["app"]
	if ok {
		appName = app.(string)
		return
	}

	return
}

func (l *License) SetAppName(appName string) {
	l.Headers["app"] = appName
}

// GetAlg returns alg defined in the license header.
func (l *License) GetAlg() (alg string) {
	algInt, ok := l.Headers["alg"]
	if ok {
		alg = algInt.(string)
		return
	}

	return alg
}

func (l *License) GetApp(appName string) (*config.App, error) {
	app, ok := config.Global.Apps[appName]
	if !ok {
		return nil, errors.New("app not found with given name")
	}

	return app, nil
}

func (l *License) ApplyApp() error {
	var alg string
	var key config.Key
	emptyKeys := config.Key{}

	appName := l.GetAppName()
	if appName == "" {
		alg = l.GetAlg()
		key = l.Key
	} else {
		app, err := l.GetApp(appName)
		if err != nil {
			return err
		}

		alg = app.Alg
		key = app.Key
	}

	if alg == "" {
		alg = "HS256"
	}

	if l.Headers == nil {
		l.Headers = make(map[string]interface{})
	}

	l.Headers["alg"] = alg

	if reflect.DeepEqual(key, emptyKeys) {
		key = config.Global.DefaultKey
	}

	l.Key = key

	return nil
}

func (l *License) Generate() error {

	if len(l.Headers) == 0 {
		l.Headers = make(map[string]interface{})
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod(l.GetAlg()), l.Claims)
	token.Header = l.Headers

	signedString, err := token.SignedString(l.SignKey)
	if err != nil {
		return err
	}

	l.Token = signedString

	l.ID = HexSHA256([]byte(signedString))

	return nil
}

func HexSHA256(key []byte) string {
	certSHA := sha256.Sum256(key)
	return hex.EncodeToString(certSHA[:])
}

func (l *License) EncryptKeys() error {

	/*switch l.signKey.(type) {
	case *rsa.PrivateKey:
		ciphertext, err := Encrypt([]byte(config.Global.AdminSecret), l.signKeyRaw)
		if err != nil {
			return err
		}

		l.Keys. = string(ciphertext)
	}
	if l.signKey == l.Keys.HMACSecret {
		ciphertext, err := Encrypt([]byte(config.Global.AdminSecret), []byte(l.Keys.HMACSecret))
		if err != nil {
			return err
		}

		l.Keys.HMACSecret = string(ciphertext)
	} else {

	}*/
	return nil

}

func (l *License) IsLicenseValid(tokenString string) (bool, error) {
	if !l.Active {
		return false, nil
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		switch token.Method.(type) {
		case *jwt.SigningMethodHMAC:
			return l.VerifyKey, nil
		case *jwt.SigningMethodRSA:

			return l.VerifyKey, nil
		default:
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
	})

	return token.Valid, err
}
