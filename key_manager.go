package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"github.com/furkansenharputlu/f-license/config"
	"net/http"
	"os"
	"strings"

	"github.com/furkansenharputlu/f-license/lcs"

	"github.com/furkansenharputlu/f-license/storage"
	"github.com/sirupsen/logrus"
)

type KeyManager struct {
}

const InternalServerError = "internal error"

func (m *KeyManager) GetOrAddKey(k *config.Key, dryRun bool) (string, int, error) {
	if k == nil {
		return "", 0, errors.New("key is nil")
	}

	getting := false
	keyID := ""

	// TODO check for private and public keys match, otherwise raise error
	if k.ID != "" && !dryRun {
		logrus.Debugf("Key will be used with the given ID %s", k.ID)

		if config.Global.LoadProductsFromDB {
			err := storage.SQLHandler.Get(k, "id = ?", k.ID)
			if err != nil {
				logrus.WithError(err).Errorf("Key with the given ID couldn't be retrieved: %s", k.ID)
				return "", http.StatusNotFound, err
			}
		} else {
			for _, product := range config.Global.Products {
				if product.Key.ID == k.ID {
					k = product.Key
					break
				}
			}
		}

		getting = true
	}

	logrus.Debug("Raw key will be used")
	var err error

	switch k.Type {
	case "hmac":
		k.Private = ""
		k.Public = ""
		if getting {
			rawHMACSecret, err := Decrypt([]byte(config.Global.Secret), []byte(k.HMAC))
			if err != nil {
				logrus.WithError(err).Error("HMAC secret couldn't be decrypted")
				return "", http.StatusInternalServerError, errors.New(InternalServerError)
			}

			k.HMAC = string(rawHMACSecret)
		} else {
			if k.HMAC == "" {
				if k.HMACPath == "" {
					return "", http.StatusBadRequest, errors.New("neither raw key nor key file path provided for hmac")
				}

				rawKeyInBytes, err := os.ReadFile(k.HMACPath)
				if err != nil {
					return "", http.StatusNotFound, err
				}

				k.HMAC = string(rawKeyInBytes)
				k.HMACPath = ""
			}

			// HMAC secret
			encrypted, err := Encrypt([]byte(config.Global.Secret), []byte(k.HMAC))
			if err != nil {
				logrus.Error("Raw hmac secret couldn't be encrypted")
				return "", http.StatusInternalServerError, err
			}

			keyID += k.HMAC

			k.HMAC = string(encrypted)
		}

	case "rsa":
		/*if rsa == nil {
			return "", http.StatusBadRequest, errors.New("key type is rsa but no rsa is set")
		}*/

		k.HMAC = "" // if it is rsa, set hmac as empty to omit in marshalling

		// RSA Private Key
		if getting {
			rawPrivateKey, err := Decrypt([]byte(config.Global.Secret), []byte(k.Private))
			if err != nil {
				logrus.WithError(err).Error("RSA private key couldn't be decrypted")
				return "", http.StatusInternalServerError, errors.New(InternalServerError)
			}

			k.Private = string(rawPrivateKey)
		} else {
			if k.Private == "" {
				if k.PrivatePath == "" {
					return "", http.StatusBadRequest, errors.New("neither raw key nor key file path provided for private key")
				}

				rawKeyInBytes, err := os.ReadFile(k.PrivatePath)
				if err != nil {
					return "", http.StatusNotFound, err
				}

				k.Private = string(rawKeyInBytes)
				k.PrivatePath = ""
			}

			// Private key
			encrypted, err := Encrypt([]byte(config.Global.Secret), []byte(k.Private))
			if err != nil {
				logrus.Error("Raw private key couldn't be decrypted")
				return "", http.StatusInternalServerError, err
			}

			keyID += k.Private

			k.Private = string(encrypted)
		}

		// RSA Public Key
		if getting {
			rawPublicKey, err := Decrypt([]byte(config.Global.Secret), []byte(k.Public))
			if err != nil {
				logrus.Error("Raw public key couldn't be decrypted")
				return "", http.StatusInternalServerError, err
			}

			k.Public = string(rawPublicKey)
		} else {
			if k.Public == "" {
				if k.PublicPath == "" {
					return "", http.StatusBadRequest, errors.New("neither raw key or key file path provided for public key")
				}

				rawKeyInBytes, err := os.ReadFile(k.PublicPath)
				if err != nil {
					return "", http.StatusNotFound, err
				}

				k.Public = string(rawKeyInBytes)
			}

			// Public key
			encrypted, err := Encrypt([]byte(config.Global.Secret), []byte(k.Public))
			if err != nil {
				logrus.Error("Raw public key couldn't be encrypted")
				return "", http.StatusInternalServerError, err
			}

			keyID += k.Public

			k.Public = string(encrypted)
		}
	default:
		return "", http.StatusBadRequest, errors.New("key type is undefined")
	}

	if !getting {
		k.ID = lcs.HexSHA256([]byte(keyID))

		if !dryRun {
			if err = storage.SQLHandler.AddIfNotExisting(k); err != nil {
				logrus.WithError(err).Debug("Couldn't store key inside license object")
				return "", http.StatusInternalServerError, err
			}
		}
	}

	return k.ID, http.StatusOK, nil
}

func getOrAddKey(getting bool, key *string, path string) (string, error, int) {
	var keyID string

	if getting {
		decryptedKey, err := Decrypt([]byte(config.Global.Secret), []byte(*key))
		if err != nil {
			return "", errors.New("couldn't decrypt"), http.StatusInternalServerError
		}

		*key = string(decryptedKey)
	} else {
		if *key == "" {
			if path == "" {
				return "", errors.New("neither raw key nor key file path provided for private key"), http.StatusBadRequest
			}

			rawKeyInBytes, err := os.ReadFile(path)
			if err != nil {
				return "", err, http.StatusNotFound
			}

			*key = string(rawKeyInBytes)
		}

		encrypted, err := Encrypt([]byte(config.Global.Secret), []byte(*key))
		if err != nil {
			logrus.Error("Raw private key couldn't be decrypted")
			return "", errors.New("couldn't encrypt"), http.StatusInternalServerError
		}

		*key = string(encrypted)

		keyID += *key
	}

	return keyID, nil, 0
}

// https://itnext.io/encrypt-data-with-a-password-in-go-b5366384e291
func Encrypt(key, data []byte) ([]byte, error) {
	key = rightPad2Len(string(key), "=", 32)
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// https://itnext.io/encrypt-data-with-a-password-in-go-b5366384e291
func Decrypt(key, data []byte) ([]byte, error) {
	key = rightPad2Len(string(key), "=", 32)
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, err
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func rightPad2Len(s, padStr string, overallLen int) []byte {
	padCountInt := 1 + (overallLen-len(padStr))/len(padStr)
	retStr := s + strings.Repeat(padStr, padCountInt)
	return []byte(retStr[:overallLen])
}
