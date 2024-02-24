package lcs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/furkansenharputlu/f-license/config"
	"github.com/furkansenharputlu/f-license/storage"
	"github.com/iancoleman/orderedmap"
	"gorm.io/gorm"
	"reflect"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type License struct {
	LicenseInfo
	KeyID string      `json:"key"`
	Key   *config.Key `json:"-"`

	SignKey   interface{} `json:"-" gorm:"-"`
	VerifyKey interface{} `json:"-" gorm:"-"`
}

type LicenseInfo struct {
	ID        string                 `json:"id"`
	Active    bool                   `json:"active"`
	Headers   *orderedmap.OrderedMap `json:"headers" gorm:"-"`
	Claims    *orderedmap.OrderedMap `json:"claims" gorm:"-"`
	Token     string                 `json:"token"`
	CreatedAt time.Time              `json:"created_at"`
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
	product, ok := l.Headers.Get("product")
	if ok {
		appName = product.(string)
		return
	}

	return
}

func (l *License) SetProductName(appName string) {
	l.Headers.Set("product", appName)
}

// GetAlg returns alg defined in the license header.
func (l *License) GetAlg() (alg string) {
	algInt, ok := l.Headers.Get("alg")
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
	/*t, _ := jwt.Parse(l.Token, nil)

	//l.Headers = t.Header
	l.Claims = t.Claims.(jwt.MapClaims)*/
}

func (l *License) ApplyProduct() error {
	var alg string
	var key *config.Key
	emptyKeys := config.Key{}

	if l.Headers == nil {
		l.Headers = orderedmap.New()
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

		if p, ok := l.Headers.Get("plan"); ok {
			plan := getPlan(p.(string))
			for _, h := range plan.Policy.Headers {
				l.Headers.Set(h.Key, h.Value)
			}

			for _, h := range plan.Policy.Claims {
				l.Claims.Set(h.Key, h.Value)
			}

			l.Claims.Set("exp", time.Now().AddDate(0, 0, plan.Policy.Expiration).Unix())
		}
	}

	if alg == "" {
		alg = "HS256"
	}

	l.Headers.Set("alg", alg)

	if reflect.DeepEqual(key, emptyKeys) {
		key = &config.Global.DefaultKey
	}

	l.Key = key

	return nil
}

type Token struct {
	*jwt.Token
	Headers *orderedmap.OrderedMap
	Claims  *orderedmap.OrderedMap
}

// SigningString is the customized version jwt.Token.SigningString.
func (t *Token) SigningString() (string, error) {
	var err error
	parts := make([]string, 2)
	for i, _ := range parts {
		var jsonValue []byte
		if i == 0 {
			if jsonValue, err = json.Marshal(t.Headers); err != nil {
				return "", err
			}
		} else {
			if jsonValue, err = json.Marshal(t.Claims); err != nil {
				return "", err
			}
		}

		parts[i] = jwt.EncodeSegment(jsonValue)
	}
	return strings.Join(parts, "."), nil
}

// SignedString is the customized version jwt.Token.SignedString.
func (t *Token) SignedString(key interface{}) (string, error) {
	var sig, sstr string
	var err error
	if sstr, err = t.SigningString(); err != nil {
		return "", err
	}
	if sig, err = t.Method.Sign(sstr, key); err != nil {
		return "", err
	}
	return strings.Join([]string{sstr, sig}, "."), nil
}

// NewWithClaims is the customized version jwt.Token.
func NewWithClaims(method jwt.SigningMethod, claims *orderedmap.OrderedMap) *Token {
	headers := orderedmap.New()
	headers.Set("typ", "JWT")
	headers.Set("alg", method.Alg())
	return &Token{
		Headers: headers,
		Claims:  claims,
		Token: &jwt.Token{
			Method: method,
		},
	}
}

func (l *License) Generate() error {

	if l.Headers == nil {
		l.Headers = orderedmap.New()
	}

	if _, ok := l.Claims.Get("iat"); !ok {
		l.Claims.Set("iat", time.Now().Unix())
	}

	token := NewWithClaims(jwt.GetSigningMethod(l.GetAlg()), l.Claims)
	token.Headers = l.Headers

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
	return hex.EncodeToString(certSHA[:3])
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
