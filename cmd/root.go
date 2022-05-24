// VulcanizeDB
// Copyright © 2022 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"fmt"
	"io"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ipld-ethcl-indexer",
	Short: "This application will keep track of all BeaconState's and SignedBeaconBlock's on the Beacon Chain.",
	Long: `This is an application that will capture the BeaconState's and SignedBeaconBlock's on the Beacon Chain.
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
		fmt.Println("Err when executing rootCmd", err)
		os.Exit(1)
	}
}

// Prerun for Cobra
func initFuncs(cmd *cobra.Command, args []string) {
	logFormat()
	if err := logFile(); err != nil {
		log.WithField("err", err).Error("Could not set log file")
	}
	if err := logLevel(); err != nil {
		log.WithField("err", err).Error("Could not set log level")
	}
}

// Set the log level for the application
func logLevel() error {
	err := viper.BindEnv("log.level", "LOGRUS_LEVEL")
	if err != nil {
		return err
	}
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

// Create a log file
func logFile() error {
	err := viper.BindEnv("log.file", "LOGRUS_FILE")
	if err != nil {
		return err
	}
	logfile := viper.GetString("log.file")
	if logfile != "" {
		file, err := os.OpenFile(logfile,
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			if viper.GetBool("log.output") {
				log.Infof("Directing output to %s", logfile)
				mw := io.MultiWriter(os.Stdout, file)
				log.SetOutput(mw)
			} else {
				log.SetOutput(file)
			}
		} else {
			log.SetOutput(os.Stdout)
			log.Info("Failed to log to file, using default stdout")
		}
	} else {
		log.SetOutput(os.Stdout)
	}
	return nil
}

// Format the logger
func logFormat() {
	logFormat := viper.GetString("log.format")

	if logFormat == "json" {
		log.SetFormatter(&log.JSONFormatter{})

	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// Optional Flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ipld-ethcl-indexer.yaml)")
	rootCmd.PersistentFlags().String("log.level", log.InfoLevel.String(), "log level (trace, debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().String("log.file", "ipld-ethcl-indexer.log", "file path for logging")
	rootCmd.PersistentFlags().Bool("log.output", true, "Should we log to STDOUT")
	rootCmd.PersistentFlags().String("log.format", "json", "json or text")

	// Bind Flags with Viper
	// Optional
	err := viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log.level"))
	exitErr(err)
	err = viper.BindPFlag("log.file", rootCmd.PersistentFlags().Lookup("log.file"))
	exitErr(err)
	err = viper.BindPFlag("log.output", rootCmd.PersistentFlags().Lookup("log.output"))
	exitErr(err)
	err = viper.BindPFlag("log.format", rootCmd.PersistentFlags().Lookup("log.format"))
	exitErr(err)

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
