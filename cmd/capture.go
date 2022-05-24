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
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dbUsername                 string
	dbPassword                 string
	dbName                     string
	dbAddress                  string
	dbDriver                   string
	dbPort                     int
	bcAddress                  string
	bcPort                     int
	bcBootRetryInterval        int
	bcBootMaxRetry             int
	bcConnectionProtocol       string
	bcType                     string
	bcMaxHistoricProcessWorker int
	kgMaxWorker                int
	kgTableIncrement           int
	kgProcessGaps              bool
	maxWaitSecondsShutdown     time.Duration  = time.Duration(20) * time.Second
	notifierCh                 chan os.Signal = make(chan os.Signal, 1)
	testDisregardSync          bool
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
	//err := captureCmd.MarkPersistentFlagRequired("db.username")
	// exitErr(err)
	// err = captureCmd.MarkPersistentFlagRequired("db.password")
	// exitErr(err)
	// err = captureCmd.MarkPersistentFlagRequired("db.address")
	// exitErr(err)
	// err = captureCmd.MarkPersistentFlagRequired("db.port")
	// exitErr(err)
	// err = captureCmd.MarkPersistentFlagRequired("db.name")
	// exitErr(err)
	// err = captureCmd.MarkPersistentFlagRequired("db.driver")
	// exitErr(err)

	//// Beacon Client Specific
	captureCmd.PersistentFlags().StringVarP(&bcAddress, "bc.address", "l", "", "Address to connect to beacon node (required)")
	captureCmd.PersistentFlags().StringVarP(&bcType, "bc.type", "", "lighthouse", "The beacon client we are using, options are prysm and lighthouse.")
	captureCmd.PersistentFlags().IntVarP(&bcPort, "bc.port", "r", 0, "Port to connect to beacon node (required )")
	captureCmd.PersistentFlags().StringVarP(&bcConnectionProtocol, "bc.connectionProtocol", "", "http", "protocol for connecting to the beacon node.")
	captureCmd.PersistentFlags().IntVarP(&bcBootRetryInterval, "bc.bootRetryInterval", "", 30, "The amount of time to wait between retries while booting the application")
	captureCmd.PersistentFlags().IntVarP(&bcBootMaxRetry, "bc.bootMaxRetry", "", 5, "The amount of time to wait between retries while booting the application")
	captureCmd.PersistentFlags().IntVarP(&bcMaxHistoricProcessWorker, "bc.maxHistoricProcessWorker", "", 30, "The number of workers that should be actively processing slots from the ethcl.historic_process table. Be careful of system memory.")
	// err = captureCmd.MarkPersistentFlagRequired("bc.address")
	// exitErr(err)
	// err = captureCmd.MarkPersistentFlagRequired("bc.port")
	// exitErr(err)

	//// Known Gaps specific
	captureCmd.PersistentFlags().BoolVarP(&kgProcessGaps, "kg.processKnownGaps", "", true, "Should we process the slots within the ethcl.known_gaps table.")
	captureCmd.PersistentFlags().IntVarP(&kgTableIncrement, "kg.increment", "", 10000, "The max slots within a single entry to the known_gaps table.")
	captureCmd.PersistentFlags().IntVarP(&kgMaxWorker, "kg.maxKnownGapsWorker", "", 30, "The number of workers that should be actively processing slots from the ethcl.known_gaps table. Be careful of system memory.")

	//// Testing Specific
	captureCmd.PersistentFlags().BoolVar(&testDisregardSync, "t.skipSync", false, "Should we disregard the head sync?")

	// Bind Flags with Viper
	//// DB Flags
	err := viper.BindPFlag("db.username", captureCmd.PersistentFlags().Lookup("db.username"))
	exitErr(err)
	err = viper.BindPFlag("db.password", captureCmd.PersistentFlags().Lookup("db.password"))
	exitErr(err)
	err = viper.BindPFlag("db.address", captureCmd.PersistentFlags().Lookup("db.address"))
	exitErr(err)
	err = viper.BindPFlag("db.port", captureCmd.PersistentFlags().Lookup("db.port"))
	exitErr(err)
	err = viper.BindPFlag("db.name", captureCmd.PersistentFlags().Lookup("db.name"))
	exitErr(err)

	//// Testing Specific
	err = viper.BindPFlag("t.skipSync", captureCmd.PersistentFlags().Lookup("t.skipSync"))
	exitErr(err)

	//// LH specific
	err = viper.BindPFlag("bc.address", captureCmd.PersistentFlags().Lookup("bc.address"))
	exitErr(err)
	err = viper.BindPFlag("bc.type", captureCmd.PersistentFlags().Lookup("bc.type"))
	exitErr(err)
	err = viper.BindPFlag("bc.port", captureCmd.PersistentFlags().Lookup("bc.port"))
	exitErr(err)
	err = viper.BindPFlag("bc.connectionProtocol", captureCmd.PersistentFlags().Lookup("bc.connectionProtocol"))
	exitErr(err)
	err = viper.BindPFlag("bc.bootRetryInterval", captureCmd.PersistentFlags().Lookup("bc.bootRetryInterval"))
	exitErr(err)
	err = viper.BindPFlag("bc.bootMaxRetry", captureCmd.PersistentFlags().Lookup("bc.bootMaxRetry"))
	exitErr(err)
	err = viper.BindPFlag("bc.maxHistoricProcessWorker", captureCmd.PersistentFlags().Lookup("bc.maxHistoricProcessWorker"))
	exitErr(err)
	// Here you will define your flags and configuration settings.

	//// Known Gap Specific
	err = viper.BindPFlag("kg.processKnownGaps", captureCmd.PersistentFlags().Lookup("kg.processKnownGaps"))
	exitErr(err)
	err = viper.BindPFlag("kg.increment", captureCmd.PersistentFlags().Lookup("kg.increment"))
	exitErr(err)
	err = viper.BindPFlag("kg.processKnownGaps", captureCmd.PersistentFlags().Lookup("kg.maxKnownGapsWorker"))
	exitErr(err)

}

// Helper function to catch any errors.
// We need to capture these errors for the linter.
func exitErr(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}
