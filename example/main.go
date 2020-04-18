package main

import (
	"f-license/fclient"
	"fmt"
	"time"
)

// make the license configurable
var license = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhZGRyZXNzIjoiSXN0YW5idWwsIFR1cmtleSIsInVzZXJuYW1lIjoiRnVya2FuIn0.Jn6tAhXlUpIrbYhfCFfiLJFgAVvMUlUnRuHurQbFzThx2VLaFcxmvhxUAs0EKYjsZqdV6QmFf5WF6mj8z_V7j0FcbFCc2kgNDZm8cH6OiCTCbaxk-JjmVCJL_rXeHPxzq56vNO0TM5f6SA0OrDSH6DxrVxLlIYZM-U52aDpzbfRZMITVz2QZ1Yth9s-FlqODwKLoZhnxslti1h2vCJDwsRyHCnNhrjPK6IYTn0y_fXlnONw2h4rTrb1ymqtN_an0Drk0rGjL8bViG1Y5tSnkM-6W0Cx9I0gLB2_5d01t2DKRoNMJ_t8clZuKRmBaZ7qSTw-pVQ5NvI1Iqq27PnAPYw"

// pin this to your app
const secret = `-----BEGIN CERTIFICATE-----
MIIDZjCCAk4CCQDIwQ3PDtNR6DANBgkqhkiG9w0BAQsFADB1MQswCQYDVQQGEwJS
UzEPMA0GA1UECAwGU2VyYmlhMREwDwYDVQQHDAhCZWxncmFkZTEMMAoGA1UECgwD
VHlrMRkwFwYDVQQLDBB3d3cudHlrLXRlc3QuY29tMRkwFwYDVQQDDBB3d3cudHlr
LXRlc3QuY29tMB4XDTE3MTExMzIzMzYyMVoXDTI3MTExMTIzMzYyMVowdTELMAkG
A1UEBhMCUlMxDzANBgNVBAgMBlNlcmJpYTERMA8GA1UEBwwIQmVsZ3JhZGUxDDAK
BgNVBAoMA1R5azEZMBcGA1UECwwQd3d3LnR5ay10ZXN0LmNvbTEZMBcGA1UEAwwQ
d3d3LnR5ay10ZXN0LmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB
ALLYxGwCzmDiec5XsynEbFZRcpdifIsEq2+Y1GbG7sacs11SeaBHBUoPnQ93ooSR
+RV+oPmZOqpvyCiSf4bIER4W4dQ1d2pE4KgPipuhGSd0T1IESbP6j/jCrY/5AM1E
SjdNMSeyWtsI/gfuhdfwX0xOXcfjI1++ySW6VbxKqoHakFtf8zRB7020kAXi5fwc
k0OgxAuZ+9SuwmyMLG3lC2Hgt0P4MgYxkNQ8YBfjSqdcZWmTlnuyvm48X/KzbJaD
fw1wgND+jIQuNw5oooFocw6pk3TV5VL8l9yQjVPK6GJRg2D1Vx+0x8Lr24ljBYWF
uvet7+4SeQp1n2kMSyCCTYkCAwEAATANBgkqhkiG9w0BAQsFAAOCAQEAJqhrlW6Z
7cW7fwxNuoOfltt/m/4HoYjDiXd0N8tkg0yOMEkLuzrb00htXA/8mVRE8YjEuh90
GW2uTCOToyeInJGbsKwxUQXX/laE8hZn8TnVOwe0cEDWeV86a83u1q82+PrscFd3
ogbyzmx6ome3drP5UOL3wCdsmbHxFBX6N3Ys889ORv+XbH1Hbre5X/ntVI0C2Cdu
Y5yBCsSOwPr5WJ9INP0eCIqKk72H8PWjeCp7PLL4iFcfGgFAeGUA0JioAPZwwO61
tKiypA6uHK3dBS9Mh5sj9VdsdxZpNh8lkSePsqu5A6Owi5WgJF0hKIOq7kv2ZbAd
aglbxDAYiMqXkg==
-----END CERTIFICATE-----
`

var LicenseValid bool

func main() {
	//localVerification()
	remoteVerification()

	if LicenseValid {
		fmt.Println("An operation can be done if the license is valid")
	}
}

func localVerification() {
	for {
		LicenseValid, err := fclient.VerifyLocally(license, secret)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println(LicenseValid)
		time.Sleep(2 * time.Second)
	}
}

func remoteVerification() {
	for {
		LicenseValid, err := fclient.VerifyRemotely("http://localhost:4242", license)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println(LicenseValid)
		time.Sleep(2 * time.Second)
	}
}
