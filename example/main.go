package main

import (
	"fmt"
	"time"

	"github.com/furkansenharputlu/f-license/client"
)

// make the license configurable
var licenseKey = "eyJhMyI6ImE0IiwiYWhtZXQgIjoic2VuIiwiYWxnIjoiSFMzODQiLCJwbGFuIjoicGxhbjEiLCJwcm9kdWN0IjoiZnVya2FuIn0.eyJiMSI6ImIyIiwiZXhwIjoxNzA5NDMwODM3LCJpYXQiOjE3MDc4NzU2Mzd9.R5d87Ox05KMc9LVapRsuGhfCA3aO-6BgJkiybMkmxG_kq2BRd4131fn-BHDMyQO1"

// pin this to your product
const secret = `my hmac secret`

var LicenseValid bool

func main() {
	localVerification()
	//remoteVerification()

	if LicenseValid {
		fmt.Println("An operation can be done if the license is valid")
	}
}

func localVerification() {
	for {
		LicenseValid, err := client.VerifyLocally(secret, licenseKey)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println(LicenseValid)
		time.Sleep(2 * time.Second)
	}
}

func remoteVerification() {
	for {
		LicenseValid, err := client.VerifyRemotely("https://localhost:4242", secret, licenseKey)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println(LicenseValid)
		time.Sleep(2 * time.Second)
	}
}
