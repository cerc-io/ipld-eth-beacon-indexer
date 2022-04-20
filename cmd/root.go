/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile        string
	dbUsername     string
	dbPassword     string
	dbName         string
	dbAddress      string
	dbDriver       string
	dbPort         int
	lhAddress      string
	lhPort         uint16
	logWithCommand log.Entry
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ipld-ethcl-indexer",
	Short: "This application will keep track of all BeaconState's and SginedBeaconBlock's on the Beacon Chain.",
	Long: `This is an application that will capture the BeaconState's and SginedBeaconBlock's on the Beacon Chain.
It can either do this will keeping track of head, or backfilling historic data.`,
	PersistentPreRun: initFuncs,
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

// Prerun for Cobra
func initFuncs(cmd *cobra.Command, args []string) {
	viper.BindEnv("log.file", "LOGRUS_FILE")
	logfile := viper.GetString("log.file")
	if logfile != "" {
		file, err := os.OpenFile(logfile,
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.Infof("Directing output to %s", logfile)
			log.SetOutput(file)
		} else {
			log.SetOutput(os.Stdout)
			log.Info("Failed to log to file, using default stdout")
		}
	} else {
		log.SetOutput(os.Stdout)
	}
	if err := logLevel(); err != nil {
		log.Fatal("Could not set log level: ", err)
	}
}

// Set the log level for the application
func logLevel() error {
	viper.BindEnv("log.level", "LOGRUS_LEVEL")
	lvl, err := log.ParseLevel(viper.GetString("log.level"))
	if err != nil {
		return err
	}
	log.SetLevel(lvl)
	if lvl > log.InfoLevel {
		log.SetReportCaller(true)
	}
	log.Info("Log level set to ", lvl.String())
	return nil
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// Optional Flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ipld-ethcl-indexer.yaml)")
	rootCmd.PersistentFlags().String("log-level", log.InfoLevel.String(), "log level (trace, debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().String("log-file", "ipld-ethcl-indexer.log", "file path for logging")

	// Required Flags

	//// DB Specific
	rootCmd.PersistentFlags().StringVarP(&dbUsername, "db.username", "", "", "Database username (required)")
	rootCmd.PersistentFlags().StringVarP(&dbPassword, "db.password", "", "", "Database Password (required)")
	rootCmd.PersistentFlags().StringVarP(&dbAddress, "db.address", "", "", "Port to connect to DB(required)")
	rootCmd.PersistentFlags().StringVarP(&dbName, "db.name", "n", "", "Database name connect to DB(required)")
	rootCmd.PersistentFlags().StringVarP(&dbDriver, "db.driver", "", "", "Database Driver to connect to DB(required)")
	rootCmd.PersistentFlags().IntVarP(&dbPort, "db.port", "", 0, "Port to connect to DB(required)")
	rootCmd.MarkPersistentFlagRequired("db.username")
	rootCmd.MarkPersistentFlagRequired("db.password")
	rootCmd.MarkPersistentFlagRequired("db.address")
	rootCmd.MarkPersistentFlagRequired("db.port")
	rootCmd.MarkPersistentFlagRequired("db.name")
	rootCmd.MarkPersistentFlagRequired("db.driver")

	//// Lighthouse Specific
	rootCmd.PersistentFlags().StringVarP(&lhAddress, "lh.address", "l", "", "Address to connect to lighthouse node (required if username is set)")
	rootCmd.PersistentFlags().Uint16VarP(&lhPort, "lh.port", "r", 0, "Port to connect to lighthouse node (required if username is set)")
	rootCmd.MarkPersistentFlagRequired("lh.address")
	rootCmd.MarkPersistentFlagRequired("lh.port")

	// Bind Flags with Viper
	// Optional
	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("log.file", rootCmd.PersistentFlags().Lookup("log-file"))

	//// DB Flags
	viper.BindPFlag("db.username", rootCmd.PersistentFlags().Lookup("db.username"))
	viper.BindPFlag("db.password", rootCmd.PersistentFlags().Lookup("db.password"))
	viper.BindPFlag("db.address", rootCmd.PersistentFlags().Lookup("db.address"))
	viper.BindPFlag("db.port", rootCmd.PersistentFlags().Lookup("db.port"))
	viper.BindPFlag("db.name", rootCmd.PersistentFlags().Lookup("db.name"))
	viper.BindPFlag("db.driver", rootCmd.PersistentFlags().Lookup("db.driver"))

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
