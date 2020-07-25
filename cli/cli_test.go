package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/furkansenharputlu/f-license/config"
	"github.com/furkansenharputlu/f-license/lcs"
	"github.com/furkansenharputlu/f-license/storage"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestMain(m *testing.M) {
	config.Global.Load("../sample_config.json")
	config.Global.DBName = "f-license_test"
	storage.Connect()
	storage.LicenseHandler.DropDatabase()
	os.Exit(m.Run())
}

func sampleLicense(lGen ...func(l *lcs.License)) (l *lcs.License) {
	l = &lcs.License{
		Active: true,
		Headers: map[string]interface{}{
			"typ": "Trial",
		},
		Claims: jwt.MapClaims{
			"name":    "Furkan",
			"address": "Istanbul, Turkey",
		},
	}

	if len(lGen) > 0 {
		lGen[0](l)
	}

	return
}

func generateLicenseJSONFile(l *lcs.License) *os.File {
	licenseBytes, _ := json.Marshal(l)
	licenseJSONFile, _ := ioutil.TempFile("", "license.json")

	_, _ = licenseJSONFile.Write(licenseBytes)

	return licenseJSONFile
}

func generateLicense(l *lcs.License) (generatedLicense map[string]string) {
	licenseJSONFile := generateLicenseJSONFile(l)
	defer licenseJSONFile.Close()

	b := bytes.NewBufferString("")
	generateCmd.SetOutput(b)
	generateCmd.SetArgs([]string{licenseJSONFile.Name()})
	_ = generateCmd.Execute()

	out, _ := ioutil.ReadAll(b)
	_ = json.Unmarshal(out, &generatedLicense)

	return generatedLicense
}

func TestGenerateCmd(t *testing.T) {
	defer storage.LicenseHandler.DropDatabase()
	l := sampleLicense()

	generatedLicense := generateLicense(l)

	assert.NotEmpty(t, generatedLicense["id"])
	assert.NotEmpty(t, generatedLicense["token"])
}

func TestVerifyCmd(t *testing.T) {
	defer storage.LicenseHandler.DropDatabase()
	l := sampleLicense()

	generatedLicense := generateLicense(l)

	verifyCmd.SetArgs([]string{generatedLicense["token"]})
	b := bytes.NewBufferString("")
	verifyCmd.SetOutput(b)
	_ = verifyCmd.Execute()

	out, _ := ioutil.ReadAll(b)

	assert.Equal(t, string(out), "true")
}

func TestActivateCmd(t *testing.T) {
	defer storage.LicenseHandler.DropDatabase()
	l := sampleLicense()

	generatedLicense := generateLicense(l)

	verifyCmd.SetArgs([]string{generatedLicense["token"]})
	b := bytes.NewBufferString("")
	verifyCmd.SetOutput(b)
	_ = verifyCmd.Execute()

	out, _ := ioutil.ReadAll(b)
	assert.Equal(t, string(out), "true")

	// Inactivate and check it is not verified
	inactivateCmd.SetArgs([]string{generatedLicense["id"]})
	_ = inactivateCmd.Execute()

	_ = verifyCmd.Execute()

	out, _ = ioutil.ReadAll(b)
	assert.Equal(t, string(out), "false")

	// Activate again and check it is verified
	activateCmd.SetArgs([]string{generatedLicense["id"]})
	_ = activateCmd.Execute()

	_ = verifyCmd.Execute()

	out, _ = ioutil.ReadAll(b)
	assert.Equal(t, string(out), "true")
}

func TestDeleteCmd(t *testing.T) {
	defer storage.LicenseHandler.DropDatabase()
	l := sampleLicense()

	generatedLicense := generateLicense(l)

	deleteCmd.SetArgs([]string{generatedLicense["id"]})
	_ = deleteCmd.Execute()

	// Shouldn't exit because it is deleted
	_ = generateCmd.Execute()
}

func TestGetCmd(t *testing.T) {
	defer storage.LicenseHandler.DropDatabase()
	l := sampleLicense(func(l *lcs.License) {
		l.Headers["alg"] = "HS512"
	})

	generatedLicense := generateLicense(l)
	id := generatedLicense["id"]
	token := generatedLicense["token"]

	setGetCMDFlags()
	b := bytes.NewBufferString("")
	getCmd.SetOutput(b)

	t.Run("Get by id", func(t *testing.T) {
		getCmd.SetArgs([]string{"--id", id})
		_ = getCmd.Execute()

		var retLicense *lcs.License
		out, _ := ioutil.ReadAll(b)
		_ = json.Unmarshal(out, &retLicense)

		l.ID, _ = primitive.ObjectIDFromHex(id)
		l.Token = token

		assert.Equal(t, l, retLicense)
	})

	t.Run("Get by token", func(t *testing.T) {
		clearFlags()
		getCmd.SetArgs([]string{"--token", token})
		_ = getCmd.Execute()

		var retLicense *lcs.License
		out, _ := ioutil.ReadAll(b)
		_ = json.Unmarshal(out, &retLicense)

		l.ID, _ = primitive.ObjectIDFromHex(id)
		l.Token = token

		assert.Equal(t, l, retLicense)
	})
}
