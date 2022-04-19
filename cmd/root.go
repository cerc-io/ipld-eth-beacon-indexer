/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	dbUserName string
	dbPassword string
	dbAddress  string
	dbPort     uint16
	lhAddress  string
	lhPort     uint16
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ipld-ethcl-indexer",
	Short: "This application will keep track of all BeaconState's and SginedBeaconBlock's on the Beacon Chain.",
	Long: `This is an application that will capture the BeaconState's and SginedBeaconBlock's on the Beacon Chain.
It can either do this will keeping track of head, or backfilling historic data.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) {},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// Optional Flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ipld-ethcl-indexer.yaml)")

	// Required Flags

	//// DB Specific
	rootCmd.PersistentFlags().StringVarP(&dbUserName, "db.username", "u", "", "Database username (required)")
	rootCmd.PersistentFlags().StringVarP(&dbPassword, "db.password", "p", "", "Database Password (required)")
	rootCmd.PersistentFlags().StringVarP(&dbAddress, "db.address", "a", "", "Port to connect to DB(required)")
	rootCmd.PersistentFlags().Uint16VarP(&dbPort, "db.port", "o", 0, "Port to connect to DB(required)")
	rootCmd.MarkPersistentFlagRequired("db.username")
	rootCmd.MarkPersistentFlagRequired("db.password")
	rootCmd.MarkPersistentFlagRequired("db.address")
	rootCmd.MarkPersistentFlagRequired("db.port")

	//// Lighthouse Specific
	rootCmd.PersistentFlags().StringVarP(&lhAddress, "lh.address", "l", "", "Address to connect to lighthouse node (required if username is set)")
	rootCmd.PersistentFlags().Uint16VarP(&lhPort, "lh.port", "r", 0, "Port to connect to lighthouse node (required if username is set)")
	rootCmd.MarkPersistentFlagRequired("lh.address")
	rootCmd.MarkPersistentFlagRequired("lh.port")

	// Bind Flags with Viper
	//// DB Flags
	viper.BindPFlag("db.username", rootCmd.PersistentFlags().Lookup("db.username"))
	viper.BindPFlag("db.password", rootCmd.PersistentFlags().Lookup("db.password"))
	viper.BindPFlag("db.address", rootCmd.PersistentFlags().Lookup("db.address"))
	viper.BindPFlag("db.port", rootCmd.PersistentFlags().Lookup("db.port"))

	// LH specific
	viper.BindPFlag("lh.address", rootCmd.PersistentFlags().Lookup("lh.address"))
	viper.BindPFlag("lh.port", rootCmd.PersistentFlags().Lookup("lh.port"))

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".ipld-ethcl-indexer" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".ipld-ethcl-indexer")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
