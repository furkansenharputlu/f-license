package main

import (
	"encoding/json"
	"errors"
	"f-license/config"
	"f-license/lcs"
	"f-license/storage"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate new license",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var l *lcs.License

		// JSON formatted license file path
		jsonFile, err := os.Open(args[0])
		checkErr(err)

		byteValue, err := ioutil.ReadAll(jsonFile)
		checkErr(err)

		err = json.Unmarshal(byteValue, &l)
		checkErr(err)

		err = l.Generate()
		checkErr(err)

		err = storage.LicenseHandler.AddIfNotExisting(l)
		checkErr(err)

		respBytes, err := json.MarshalIndent(struct {
			ID    string `json:"id"`
			Token string `json:"token"`
		}{
			ID:    l.ID.Hex(),
			Token: l.Token,
		}, "", "    ")

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), string(respBytes))
	},
}

var activateCmd = &cobra.Command{
	Use:   "activate",
	Short: "Activate license",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := storage.LicenseHandler.Activate(args[0], false)
		checkErr(err)
	},
}

var inactivateCmd = &cobra.Command{
	Use:   "inactivate",
	Short: "Inactivate license",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := storage.LicenseHandler.Activate(args[0], true)
		checkErr(err)
	},
}

var getByIDFlag string
var getByTokenFlag string

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get license",
	Run: func(cmd *cobra.Command, args []string) {
		var l lcs.License
		if getByIDFlag != "" {
			err := storage.LicenseHandler.GetByID(getByIDFlag, &l)
			logrus.Info("Passed id value: ", getByIDFlag)
			checkErr(err)
		} else if getByTokenFlag != "" {
			err := storage.LicenseHandler.GetByToken(getByTokenFlag, &l)
			logrus.Info("Passed token value: ", getByTokenFlag)
			checkErr(err)
		} else {
			checkErr(errors.New("pass id or token"))
		}

		licenseBytes, err := json.MarshalIndent(l, "", "    ")
		checkErr(err)

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), string(licenseBytes))
	},
}

func clearFlags() {
	getByIDFlag = ""
	getByTokenFlag = ""
}

func setGetCMDFlags() {
	getCmd.Flags().StringVarP(&getByIDFlag, "id", "i", "", "License ID")
	getCmd.Flags().StringVarP(&getByTokenFlag, "token", "t", "", "License token")
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete license",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := storage.LicenseHandler.DeleteByID(args[0])
		checkErr(err)
	},
}

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify license",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var l lcs.License
		err := storage.LicenseHandler.GetByToken(args[0], &l)
		checkErr(err)

		valid, err := l.IsLicenseValid(args[0])
		checkErr(err)

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%v", valid)
	},
}

var rootCmd = &cobra.Command{
	Use:   "f-cli",
	Short: "f-cli is the terminal tool for f-license",
}

func main() {
	config.Global.Load("config.json")
	storage.Connect()

	setGetCMDFlags()

	rootCmd.AddCommand(activateCmd)
	rootCmd.AddCommand(inactivateCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(verifyCmd)
	checkErr(rootCmd.Execute())
}

func checkErr(err error) {
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}
