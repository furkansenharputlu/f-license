package main

import (
	"encoding/json"
	"errors"
	"f-license/lcs"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate new license",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var l lcs.License

		// JSON formatted license file path
		jsonFile, err := os.Open(args[0])
		checkErr(err)

		byteValue, err := ioutil.ReadAll(jsonFile)
		checkErr(err)

		err = json.Unmarshal(byteValue, &l)
		checkErr(err)

		err = l.Add()
		checkErr(err)

		fmt.Println("ID:", l.ID.Hex())
		fmt.Println("Token:", l.Token)
	},
}

var activateCmd = &cobra.Command{
	Use:   "activate",
	Short: "Activate license",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var l lcs.License
		err := l.Activate(args[0], false)
		checkErr(err)
	},
}

var inactivateCmd = &cobra.Command{
	Use:   "inactivate",
	Short: "Inactivate license",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var l lcs.License
		err := l.Activate(args[0], true)
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
			err := l.GetByID(getByIDFlag)
			logrus.Info("Passed flag value:", getByIDFlag)
			checkErr(err)
		} else if getByTokenFlag != "" {
			err := l.GetByToken(getByTokenFlag)
			logrus.Info("Passed flag value:", getByTokenFlag)
			checkErr(err)
		} else {
			checkErr(errors.New("pass id or token"))
		}

		licenseBytes, err := json.MarshalIndent(l, "", "    ")
		checkErr(err)

		fmt.Println(string(licenseBytes))
	},
}

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify license",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var l lcs.License
		err := l.GetByToken(args[0])
		checkErr(err)

		valid, err := l.IsLicenseValid(args[0])
		checkErr(err)
		fmt.Println(valid)
	},
}

var rootCmd = &cobra.Command{
	Use:   "f-cli",
	Short: "f-cli is the terminal tool for f-license",
}

func main() {
	getCmd.Flags().StringVarP(&getByIDFlag, "id", "i", "", "License ID")
	getCmd.Flags().StringVarP(&getByTokenFlag, "token", "t", "", "License token")

	rootCmd.AddCommand(activateCmd)
	rootCmd.AddCommand(inactivateCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(verifyCmd)
	checkErr(rootCmd.Execute())
}

func checkErr(err error) {
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}
