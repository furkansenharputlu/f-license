package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/furkansenharputlu/f-license/lcs"

	"github.com/furkansenharputlu/f-license/config"
	"github.com/furkansenharputlu/f-license/storage"
	"github.com/sirupsen/logrus"
)

type KeyManager struct {
}

const InternalServerError = "internal error"

func (m *KeyManager) GetOrAddKey(k *config.Key, dryRun bool) (string, int, error) {

	getting := false

	keyID := ""

	// TODO check for private and public keys match, otherwise raise error
	if k.ID != "" && !dryRun {
		logrus.Debugf("Key will be used with the given ID %s", k.ID)

		keyFound := false
		for _, app := range config.Global.Apps {
			if app.Key.ID == k.ID {
				k = &app.Key
				keyFound = true
				break
			}
		}

		if !keyFound {
			err := storage.GlobalKeyHandler.GetByID(k.ID, k)
			if err != nil {
				logrus.WithError(err).Errorf("Key with the given ID couldn't be retrieved: %s", k.ID)
				return "", http.StatusNotFound, err
			}
		}

		getting = true
	}

	logrus.Debug("Raw key will be used")
	var err error
	if k.Type == "hmac" {
		hmac := k.HMAC
		if hmac == nil {
			return "", http.StatusBadRequest, errors.New("key type is hmac but no secret is set")
		}
		k.RSA = nil // if it is hmac, set rsa as nil to omit in marshallings
		if getting {
			rawHMACSecret, err := Decrypt([]byte(config.Global.Secret), hmac.Encrypted)
			if err != nil {
				logrus.WithError(err).Error("HMAC secret couldn't be decrypted")
				return "", http.StatusInternalServerError, errors.New(InternalServerError)
			}

			hmac.Raw = string(rawHMACSecret)
		} else {
			if hmac.Raw == "" {
				if hmac.FilePath == "" {
					return "", http.StatusBadRequest, errors.New("neither raw key or key file path provided")
				}

				rawKeyInBytes, err := ioutil.ReadFile(hmac.FilePath)
				if err != nil {
					return "", http.StatusNotFound, err
				}

				hmac.Raw = string(rawKeyInBytes)
			}

			// HMAC secret
			rawHMACSecretInBytes := []byte(hmac.Raw)
			hmac.Encrypted, err = Encrypt([]byte(config.Global.Secret), rawHMACSecretInBytes)
			if err != nil {
				logrus.Error("Raw hmac secret couldn't be encrypted")
				return "", http.StatusInternalServerError, err
			}

			keyID += hmac.Raw
		}

	} else if k.Type == "rsa" {
		rsa := k.RSA
		if rsa == nil {
			return "", http.StatusBadRequest, errors.New("key type is rsa but no rsa is set")
		}

		k.HMAC = nil // if it is rsa, set hmac as nil to omit in marshallings

		// RSA Private Key
		if private := rsa.Private; private != nil {
			if getting {
				rawPrivateKey, err := Decrypt([]byte(config.Global.Secret), private.Encrypted)
				if err != nil {
					logrus.WithError(err).Error("RSA private key couldn't be decrypted")
					return "", http.StatusInternalServerError, errors.New(InternalServerError)
				}

				private.Raw = string(rawPrivateKey)
			} else {

				if private.Raw == "" {
					if private.FilePath == "" {
						return "", http.StatusBadRequest, errors.New("neither raw key nor key file path provided for private key")
					}

					rawKeyInBytes, err := ioutil.ReadFile(private.FilePath)
					if err != nil {
						return "", http.StatusNotFound, err
					}

					private.Raw = string(rawKeyInBytes)
				}

				// Private key
				rawPrivateKeyInBytes := []byte(private.Raw)
				private.Encrypted, err = Encrypt([]byte(config.Global.Secret), rawPrivateKeyInBytes)
				if err != nil {
					logrus.Error("Raw private key couldn't be decrypted")
					return "", http.StatusInternalServerError, err
				}

				keyID += private.Raw
			}
		}

		// RSA Public Key
		if public := rsa.Public; public != nil {
			if getting {
				rawPublicKey, err := Decrypt([]byte(config.Global.Secret), public.Encrypted)
				if err != nil {
					logrus.Error("Raw public key couldn't be decrypted")
					return "", http.StatusInternalServerError, err
				}

				public.Raw = string(rawPublicKey)
			} else {
				if public.Raw == "" {
					if public.FilePath == "" {
						return "", http.StatusBadRequest, errors.New("neither raw key or key file path provided for public key")
					}

					rawKeyInBytes, err := ioutil.ReadFile(public.FilePath)
					if err != nil {
						return "", http.StatusNotFound, err
					}

					public.Raw = string(rawKeyInBytes)
				}

				// Public key
				rawPublicKeyInBytes := []byte(public.Raw)
				public.Encrypted, err = Encrypt([]byte(config.Global.Secret), rawPublicKeyInBytes)
				if err != nil {
					logrus.Error("Raw public key couldn't be encrypted")
					return "", http.StatusInternalServerError, err
				}

				keyID += public.Raw
			}
		}
	} else {
		return "", http.StatusBadRequest, errors.New("key type is undefined")
	}

	if !getting {
		k.ID = lcs.HexSHA256([]byte(keyID))

		if !dryRun {
			err = storage.GlobalKeyHandler.AddIfNotExisting(k)
			if err != nil {
				logrus.WithError(err).Debug("Couldn't store key inside license object")

				return "", http.StatusInternalServerError, err
			}
		}
	}

	return k.ID, http.StatusOK, nil
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
