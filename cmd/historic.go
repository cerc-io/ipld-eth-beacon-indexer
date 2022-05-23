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
	"github.com/vulcanize/ipld-ethcl-indexer/internal/boot"
	"github.com/vulcanize/ipld-ethcl-indexer/internal/shutdown"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

// historicCmd represents the historic command
var historicCmd = &cobra.Command{
	Use:   "historic",
	Short: "Capture the historic blocks and states.",
	Long:  `Capture the historic blocks and states.`,
	Run: func(cmd *cobra.Command, args []string) {
		startHistoricProcessing()
	},
}

// Start the application to process historical slots.
func startHistoricProcessing() {
	// Boot the application
	log.Info("Starting the application in head tracking mode.")
	ctx := context.Background()

	Bc, Db, err := boot.BootApplicationWithRetry(ctx, dbAddress, dbPort, dbName, dbUsername, dbPassword, dbDriver,
		bcAddress, bcPort, bcConnectionProtocol, bcType, bcBootRetryInterval, bcBootMaxRetry, kgTableIncrement, "historic", testDisregardSync)
	if err != nil {
		StopApplicationPreBoot(err, Db)
	}
	errs := Bc.CaptureHistoric(2)
	if errs != nil {
		log.WithFields(log.Fields{
			"TotalErrors": errs,
		}).Error("The historical processing service ended after receiving too many errors.")
	}

	// Shutdown when the time is right.
	err = shutdown.ShutdownServices(ctx, notifierCh, maxWaitSecondsShutdown, Db, Bc)
	if err != nil {
		loghelper.LogError(err).Error("Ungracefully Shutdown ipld-ethcl-indexer!")
	} else {
		log.Info("Gracefully shutdown ipld-ethcl-indexer")
	}
}

func init() {
	captureCmd.AddCommand(historicCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// historicCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// historicCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// Stop the application during its initial boot phases.
func StopApplicationPreBoot(startErr error, db sql.Database) {
	loghelper.LogError(startErr).Error("Unable to Start application")
	if db != nil {
		db.Close()
	}
	os.Exit(1)
}
