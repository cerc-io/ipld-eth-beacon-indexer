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
	"context"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vulcanize/ipld-ethcl-indexer/internal/boot"
	"github.com/vulcanize/ipld-ethcl-indexer/internal/shutdown"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

var (
	kgTableIncrement int
)

// headCmd represents the head command
var headCmd = &cobra.Command{
	Use:   "head",
	Short: "Capture only the blocks and state at head.",
	Long:  `Capture only the blocks and state at head.`,
	Run: func(cmd *cobra.Command, args []string) {
		startHeadTracking()
	},
}

// Start the application to track at head.
func startHeadTracking() {
	// Boot the application
	log.Info("Starting the application in head tracking mode.")
	ctx := context.Background()

	BC, DB, err := boot.BootApplicationWithRetry(ctx, dbAddress, dbPort, dbName, dbUsername, dbPassword, dbDriver, bcAddress, bcPort, bcConnectionProtocol, bcType, bcBootRetryInterval, bcBootMaxRetry, "historic", testDisregardSync)
	if err != nil {
		loghelper.LogError(err).Error("Unable to Start application")
		if DB != nil {
			DB.Close()
		}
		os.Exit(1)
	}

	log.Info("The Beacon Client has booted successfully!")
	// Capture head blocks
	go BC.CaptureHead(kgTableIncrement)

	// Shutdown when the time is right.
	err = shutdown.ShutdownServices(ctx, notifierCh, maxWaitSecondsShutdown, DB, BC)
	if err != nil {
		loghelper.LogError(err).Error("Ungracefully Shutdown ipld-ethcl-indexer!")
	} else {
		log.Info("Gracefully shutdown ipld-ethcl-indexer")
	}

}

func init() {
	captureCmd.AddCommand(headCmd)

	// Known Gaps specific
	captureCmd.PersistentFlags().IntVarP(&kgTableIncrement, "kg.increment", "", 10000, "The max slots within a single entry to the known_gaps table.")
	err := viper.BindPFlag("kg.increment", captureCmd.PersistentFlags().Lookup("kg.increment"))
	exitErr(err)
}
