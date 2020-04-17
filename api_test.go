package main

import (
	"encoding/json"
	"f-license/lcs"
	"f-license/storage"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateLicense(t *testing.T) {
	defer storage.LicenseHandler.DropDatabase()

	path := "/admin/licenses"

	var l lcs.License
	resp := tr.Run(t, &TestCase{Method: http.MethodPost, Path: path, Data: l, BodyMatch: `"id":.*"token":"ey.*"`})
	resBytes, _ := ioutil.ReadAll(resp.Body)

	var resMap map[string]string
	_ = json.Unmarshal(resBytes, &resMap)

	tr.Run(t, &TestCase{Method: http.MethodPost, Path: path, Data: l,
		BodyMatch: "there is already such license with ID: " + resMap["id"]})
}

func TestGetLicense(t *testing.T) {
	defer storage.LicenseHandler.DropDatabase()

	path := "/admin/licenses"

	l := sampleLicense()

	resp := tr.Run(t, &TestCase{Method: http.MethodPost, Path: path, Data: l, BodyMatch: `"id":.*"token":"ey.*"`})
	resBytes, _ := ioutil.ReadAll(resp.Body)

	var resMap map[string]string
	_ = json.Unmarshal(resBytes, &resMap)

	expectedID := resMap["id"]
	expectedToken := resMap["token"]

	getPath := "/admin/licenses/" + expectedID

	resp = tr.Run(t, &TestCase{Method: http.MethodGet, Path: getPath})
	resBytes, _ = ioutil.ReadAll(resp.Body)

	var retLicense lcs.License
	_ = json.Unmarshal(resBytes, &retLicense)

	assert.Equal(t, l.Type, retLicense.Type)
	assert.Equal(t, l.Claims, retLicense.Claims)
	assert.Equal(t, l.Active, retLicense.Active)
	assert.Equal(t, expectedID, retLicense.ID.Hex())
	assert.Equal(t, expectedToken, retLicense.Token)
}

func TestVerifyLicense(t *testing.T) {
	defer storage.LicenseHandler.DropDatabase()

	path := "/admin/licenses"

	resp := tr.Run(t, &TestCase{Method: http.MethodPost, Path: path, Data: sampleLicense(), BodyMatch: `"id":.*"token":"ey.*"`})
	resBytes, _ := ioutil.ReadAll(resp.Body)

	var resMap map[string]string
	_ = json.Unmarshal(resBytes, &resMap)

	verifyPath := "/license/verify"
	formParams := map[string]string{
		"token": resMap["token"],
	}

	tr.Run(t, &TestCase{Method: http.MethodPost, Path: verifyPath, FormParams: formParams, BodyMatch: `"valid":true`})
}

func TestDeleteLicense(t *testing.T) {
	defer storage.LicenseHandler.DropDatabase()

	path := "/admin/licenses"

	var l lcs.License
	resp := tr.Run(t, &TestCase{Method: http.MethodPost, Path: path, Data: l, BodyMatch: `"id":.*"token":"ey.*"`})
	resBytes, _ := ioutil.ReadAll(resp.Body)

	var resMap map[string]string
	_ = json.Unmarshal(resBytes, &resMap)

	expectedID := resMap["id"]

	tr.Run(t, &TestCase{Method: http.MethodPost, Path: path, Data: l,
		BodyMatch: "there is already such license with ID: " + expectedID})

	deletePath := fmt.Sprintf("/admin/licenses/%s/delete", expectedID)

	tr.Run(t, &TestCase{Method: http.MethodDelete, Path: deletePath, BodyMatch: "License successfully deleted"})

	tr.Run(t, &TestCase{Method: http.MethodPost, Path: path, Data: l, BodyMatch: `"id":.*"token":"ey.*"`})
}

func TestChangeLicenseActiveness(t *testing.T) {
	defer storage.LicenseHandler.DropDatabase()

	path := "/admin/licenses"

	l := sampleLicense()
	resp := tr.Run(t, &TestCase{Method: http.MethodPost, Path: path, Data: l, BodyMatch: `"id":.*"token":"ey.*"`})
	resBytes, _ := ioutil.ReadAll(resp.Body)

	var resMap map[string]string
	_ = json.Unmarshal(resBytes, &resMap)

	licenseID := resMap["id"]

	inactivatePath := fmt.Sprintf("/admin/licenses/%s/inactivate", licenseID)
	activatePath := fmt.Sprintf("/admin/licenses/%s/activate", licenseID)

	tr.Run(t, &TestCase{Method: http.MethodPut, Path: inactivatePath, BodyMatch: `{"message":"Inactivated"}`})
	tr.Run(t, &TestCase{Method: http.MethodPut, Path: inactivatePath, BodyMatch: `{"error":"already inactive"}`})

	tr.Run(t, &TestCase{Method: http.MethodPut, Path: activatePath, BodyMatch: `{"message":"Activated"}`})
	tr.Run(t, &TestCase{Method: http.MethodPut, Path: activatePath, BodyMatch: `{"error":"already active"}`})
}
