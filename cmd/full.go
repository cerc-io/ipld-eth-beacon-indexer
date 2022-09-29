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
	"fmt"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/beaconclient"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vulcanize/ipld-eth-beacon-indexer/internal/boot"
	"github.com/vulcanize/ipld-eth-beacon-indexer/internal/shutdown"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/loghelper"
	"golang.org/x/sync/errgroup"
)

// fullCmd represents the full command
var fullCmd = &cobra.Command{
	Use:   "full",
	Short: "Capture all components of the application (head and historical)",
	Long:  `Capture all components of the application (head and historical)`,
	Run: func(cmd *cobra.Command, args []string) {
		startFullProcessing()
	},
}

func init() {
	captureCmd.AddCommand(fullCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// fullCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// fullCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// Start the application to track at head and historical processing.
func startFullProcessing() {
	// Boot the application
	log.Info("Starting the application in head tracking mode.")
	ctx := context.Background()

	Bc, Db, err := boot.BootApplicationWithRetry(ctx, viper.GetString("db.address"), viper.GetInt("db.port"), viper.GetString("db.name"), viper.GetString("db.username"), viper.GetString("db.password"), viper.GetString("db.driver"),
		viper.GetString("bc.address"), viper.GetInt("bc.port"), viper.GetString("bc.connectionProtocol"), viper.GetString("bc.type"), viper.GetInt("bc.bootRetryInterval"), viper.GetInt("bc.bootMaxRetry"),
		viper.GetInt("kg.increment"), "head", viper.GetBool("t.skipSync"), viper.GetInt("bc.uniqueNodeIdentifier"), viper.GetBool("bc.checkDb"),
		viper.GetBool("bc.performBeaconBlockProcessing"), viper.GetBool("bc.performBeaconStateProcessing"))
	if err != nil {
		StopApplicationPreBoot(err, Db)
	}

	if viper.GetBool("pm.metrics") {
		addr := viper.GetString("pm.address") + ":" + strconv.Itoa(viper.GetInt("pm.port"))
		serveProm(addr)
	}

	log.Info("The Beacon Client has booted successfully!")
	// Capture head blocks
	go Bc.CaptureHead()

	hpContext, hpCancel := context.WithCancel(context.Background())

	errG, _ := errgroup.WithContext(context.Background())
	errG.Go(func() error {
		errs := Bc.CaptureHistoric(hpContext, viper.GetInt("bc.maxHistoricProcessWorker"), beaconclient.Slot(viper.GetUint64("bc.minimumSlot")))
		if len(errs) != 0 {
			if len(errs) != 0 {
				log.WithFields(log.Fields{"errs": errs}).Error("All errors when processing historic events")
				return fmt.Errorf("Application ended because there were too many error when attempting to process historic")
			}
		}
		return nil
	})
	kgCtx, KgCancel := context.WithCancel(context.Background())
	if viper.GetBool("kg.processKnownGaps") {
		go func() {
			errG := new(errgroup.Group)
			errG.Go(func() error {
				errs := Bc.ProcessKnownGaps(kgCtx, viper.GetInt("kg.maxKnownGapsWorker"), beaconclient.Slot(viper.GetUint64("kg.minimumSlot")))
				if len(errs) != 0 {
					log.WithFields(log.Fields{"errs": errs}).Error("All errors when processing knownGaps")
					return fmt.Errorf("Application ended because there were too many error when attempting to process knownGaps")
				}
				return nil
			})
			if err := errG.Wait(); err != nil {
				loghelper.LogError(err).Error("Error with knownGaps processing")
			}
		}()
	}

	// Shutdown when the time is right.
	err = shutdown.ShutdownFull(ctx, KgCancel, hpCancel, notifierCh, maxWaitSecondsShutdown, Db, Bc)
	if err != nil {
		loghelper.LogError(err).Error("Ungracefully Shutdown ipld-eth-beacon-indexer!")
	} else {
		log.Info("Gracefully shutdown ipld-eth-beacon-indexer")
	}

}
