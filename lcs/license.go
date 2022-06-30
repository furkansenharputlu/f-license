package lcs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/furkansenharputlu/f-license/config"
	"github.com/furkansenharputlu/f-license/storage"
	"gorm.io/gorm"
	"reflect"

	"github.com/dgrijalva/jwt-go"
)

type License struct {
	LicenseInfo
	KeyID string      `json:"keyId"`
	Key   *config.Key `json:"key,omitempty"`

	SignKey   interface{} `json:"-" gorm:"-"`
	VerifyKey interface{} `json:"-" gorm:"-"`
}

type LicenseInfo struct {
	ID      string                 `json:"id"`
	Active  bool                   `json:"active"`
	Headers map[string]interface{} `json:"headers" gorm:"-"`
	Claims  jwt.MapClaims          `json:"claims" gorm:"-"`
	Token   string                 `json:"token"`
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

func (l *License) GetProductName() (appName string) {
	product, ok := l.Headers["product"]
	if ok {
		appName = product.(string)
		return
	}

	return
}

func (l *License) SetProductName(appName string) {
	l.Headers["product"] = appName
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

func (l *License) GetProduct(name string) (*config.Product, error) {
	var product config.Product
	db := storage.SQLHandler.DB().(*gorm.DB)
	err := db.Preload("Key").Where("name = ?", name).Find(&product).Error
	if err != nil {
		return nil, err
	}

	return &product, nil
}

func (l *License) DecodeToken() {
	t, _ := jwt.Parse(l.Token, nil)

	l.Headers = t.Header
	l.Claims = t.Claims.(jwt.MapClaims)
}

func (l *License) ApplyProduct() error {
	var alg string
	var key *config.Key
	emptyKeys := config.Key{}

	if l.Headers == nil {
		l.Headers = make(map[string]interface{})
	}

	appName := l.GetProductName()
	if appName == "" {
		alg = l.GetAlg()
		key = l.Key
	} else {
		product, err := l.GetProduct(appName)
		if err != nil {
			return err
		}

		alg = product.Alg
		key = product.Key

		getPlan := func(name string) *config.Plan {
			for _, plan := range product.Plans {
				if plan.Name == name {
					return plan
				}
			}

			return nil
		}

		plan := getPlan(l.Headers["plan"].(string))

		for k, v := range plan.Policy.Headers {
			l.Headers[k] = v
		}
	}

	if alg == "" {
		alg = "HS256"
	}

	l.Headers["alg"] = alg

	if reflect.DeepEqual(key, emptyKeys) {
		key = &config.Global.DefaultKey
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
	return hex.EncodeToString(certSHA[:10])
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
