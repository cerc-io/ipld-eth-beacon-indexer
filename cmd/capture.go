/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dbUsername             string
	dbPassword             string
	dbName                 string
	dbAddress              string
	dbDriver               string
	dbPort                 int
	bcAddress              string
	bcPort                 int
	bcConnectionProtocol   string
	bcType                 string
	maxWaitSecondsShutdown time.Duration  = time.Duration(5) * time.Second
	notifierCh             chan os.Signal = make(chan os.Signal, 1)
	testDisregardSync      bool
)

// captureCmd represents the capture command
var captureCmd = &cobra.Command{
	Use:   "capture",
	Short: "Capture the SignedBeaconBlocks and BeaconStates from the Beacon Chain",
	Long: `Capture SignedBeaconBlocks and BeaconStates from the Beacon Chain.
	These blocks and states will be captured in
	Postgres. They require a beacon client to be connected. You can run this to
	capture blocks and states at head or historic blocks.`,
}

func init() {
	rootCmd.AddCommand(captureCmd)

	// Required Flags

	//// DB Specific
	captureCmd.PersistentFlags().StringVarP(&dbUsername, "db.username", "", "", "Database username (required)")
	captureCmd.PersistentFlags().StringVarP(&dbPassword, "db.password", "", "", "Database Password (required)")
	captureCmd.PersistentFlags().StringVarP(&dbAddress, "db.address", "", "", "Port to connect to DB(required)")
	captureCmd.PersistentFlags().StringVarP(&dbName, "db.name", "n", "", "Database name connect to DB(required)")
	captureCmd.PersistentFlags().StringVarP(&dbDriver, "db.driver", "", "", "Database Driver to connect to DB(required)")
	captureCmd.PersistentFlags().IntVarP(&dbPort, "db.port", "", 0, "Port to connect to DB(required)")
	err := captureCmd.MarkPersistentFlagRequired("db.username")
	exitErr(err)
	err = captureCmd.MarkPersistentFlagRequired("db.password")
	exitErr(err)
	err = captureCmd.MarkPersistentFlagRequired("db.address")
	exitErr(err)
	err = captureCmd.MarkPersistentFlagRequired("db.port")
	exitErr(err)
	err = captureCmd.MarkPersistentFlagRequired("db.name")
	exitErr(err)
	err = captureCmd.MarkPersistentFlagRequired("db.driver")
	exitErr(err)

	//// Beacon Client Specific
	captureCmd.PersistentFlags().StringVarP(&bcAddress, "bc.address", "l", "", "Address to connect to beacon node (required)")
	captureCmd.PersistentFlags().StringVarP(&bcType, "bc.type", "", "lighthouse", "The beacon client we are using, options are prysm and lighthouse.")
	captureCmd.PersistentFlags().IntVarP(&bcPort, "bc.port", "r", 0, "Port to connect to beacon node (required )")
	captureCmd.PersistentFlags().StringVarP(&bcConnectionProtocol, "bc.connectionProtocol", "", "http", "protocol for connecting to the beacon node.")
	err = captureCmd.MarkPersistentFlagRequired("bc.address")
	exitErr(err)
	err = captureCmd.MarkPersistentFlagRequired("bc.port")
	exitErr(err)

	//// Testing Specific
	captureCmd.PersistentFlags().BoolVar(&testDisregardSync, "t.skipSync", false, "Should we disregard the head sync?")

	// Bind Flags with Viper
	//// DB Flags
	err = viper.BindPFlag("db.username", captureCmd.PersistentFlags().Lookup("db.username"))
	exitErr(err)
	err = viper.BindPFlag("db.password", captureCmd.PersistentFlags().Lookup("db.password"))
	exitErr(err)
	err = viper.BindPFlag("db.address", captureCmd.PersistentFlags().Lookup("db.address"))
	exitErr(err)
	err = viper.BindPFlag("db.port", captureCmd.PersistentFlags().Lookup("db.port"))
	exitErr(err)
	err = viper.BindPFlag("db.name", captureCmd.PersistentFlags().Lookup("db.name"))
	exitErr(err)
	err = viper.BindPFlag("t.skipSync", captureCmd.PersistentFlags().Lookup("t.skipSync"))
	exitErr(err)

	// Testing Specific
	err = viper.BindPFlag("t.driver", captureCmd.PersistentFlags().Lookup("db.driver"))
	exitErr(err)

	// LH specific
	err = viper.BindPFlag("bc.address", captureCmd.PersistentFlags().Lookup("bc.address"))
	exitErr(err)
	err = viper.BindPFlag("bc.type", captureCmd.PersistentFlags().Lookup("bc.type"))
	exitErr(err)
	err = viper.BindPFlag("bc.port", captureCmd.PersistentFlags().Lookup("bc.port"))
	exitErr(err)
	err = viper.BindPFlag("bc.connectionProtocol", captureCmd.PersistentFlags().Lookup("bc.connectionProtocol"))
	exitErr(err)
	// Here you will define your flags and configuration settings.

}

// Helper function to catch any errors.
// We need to capture these errors for the linter.
func exitErr(err error) {
	if err != nil {
		os.Exit(1)
	}
}
