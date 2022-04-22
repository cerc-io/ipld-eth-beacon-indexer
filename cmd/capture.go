/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dbUsername string
	dbPassword string
	dbName     string
	dbAddress  string
	dbDriver   string
	dbPort     int
	bcAddress  string
	bcPort     int
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
	captureCmd.MarkPersistentFlagRequired("db.username")
	captureCmd.MarkPersistentFlagRequired("db.password")
	captureCmd.MarkPersistentFlagRequired("db.address")
	captureCmd.MarkPersistentFlagRequired("db.port")
	captureCmd.MarkPersistentFlagRequired("db.name")
	captureCmd.MarkPersistentFlagRequired("db.driver")

	//// Beacon Client Specific
	captureCmd.PersistentFlags().StringVarP(&bcAddress, "bc.address", "l", "", "Address to connect to beacon node (required if username is set)")
	captureCmd.PersistentFlags().IntVarP(&bcPort, "bc.port", "r", 0, "Port to connect to beacon node (required if username is set)")
	captureCmd.MarkPersistentFlagRequired("bc.address")
	captureCmd.MarkPersistentFlagRequired("bc.port")

	// Bind Flags with Viper
	//// DB Flags
	viper.BindPFlag("db.username", captureCmd.PersistentFlags().Lookup("db.username"))
	viper.BindPFlag("db.password", captureCmd.PersistentFlags().Lookup("db.password"))
	viper.BindPFlag("db.address", captureCmd.PersistentFlags().Lookup("db.address"))
	viper.BindPFlag("db.port", captureCmd.PersistentFlags().Lookup("db.port"))
	viper.BindPFlag("db.name", captureCmd.PersistentFlags().Lookup("db.name"))
	viper.BindPFlag("db.driver", captureCmd.PersistentFlags().Lookup("db.driver"))

	// LH specific
	viper.BindPFlag("bc.address", captureCmd.PersistentFlags().Lookup("bc.address"))
	viper.BindPFlag("bc.port", captureCmd.PersistentFlags().Lookup("bc.port"))
	// Here you will define your flags and configuration settings.

}
