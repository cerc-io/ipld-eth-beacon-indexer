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
	"os"
	"strconv"

	"net/http"
	_ "net/http/pprof"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vulcanize/ipld-eth-beacon-indexer/internal/boot"
	"github.com/vulcanize/ipld-eth-beacon-indexer/internal/shutdown"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/loghelper"
	"golang.org/x/sync/errgroup"
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

	Bc, Db, err := boot.BootApplicationWithRetry(ctx, viper.GetString("db.address"), viper.GetInt("db.port"), viper.GetString("db.name"), viper.GetString("db.username"), viper.GetString("db.password"), viper.GetString("db.driver"),
		viper.GetString("bc.address"), viper.GetInt("bc.port"), viper.GetString("bc.connectionProtocol"), viper.GetString("bc.type"), viper.GetInt("bc.bootRetryInterval"), viper.GetInt("bc.bootMaxRetry"),
		viper.GetInt("kg.increment"), "historic", viper.GetBool("t.skipSync"), viper.GetInt("bc.uniqueNodeIdentifier"), viper.GetBool("bc.checkDb"))
	if err != nil {
		StopApplicationPreBoot(err, Db)
	}

	if viper.GetBool("pm.metrics") {
		addr := viper.GetString("pm.address") + ":" + strconv.Itoa(viper.GetInt("pm.port"))
		serveProm(addr)
	}

	hpContext, hpCancel := context.WithCancel(context.Background())

	errG, _ := errgroup.WithContext(context.Background())
	errG.Go(func() error {
		errs := Bc.CaptureHistoric(hpContext, viper.GetInt("bc.maxHistoricProcessWorker"))
		if len(errs) != 0 {
			if len(errs) != 0 {
				log.WithFields(log.Fields{"errs": errs}).Error("All errors when processing historic events")
				return fmt.Errorf("Application ended because there were too many error when attempting to process historic")
			}
		}
		return nil
	})

	kgContext, kgCancel := context.WithCancel(context.Background())
	if viper.GetBool("kg.processKnownGaps") {
		go func() {
			errG := new(errgroup.Group)
			errG.Go(func() error {
				errs := Bc.ProcessKnownGaps(kgContext, viper.GetInt("kg.maxKnownGapsWorker"))
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

	if viper.GetBool("t.pprof") {
		go func() {
			log.Println(http.ListenAndServe(fmt.Sprint("localhost:"+strconv.Itoa(viper.GetInt("t.pprofPort"))), nil))
		}()
	}

	// Shutdown when the time is right.
	err = shutdown.ShutdownHistoricProcessing(ctx, kgCancel, hpCancel, notifierCh, maxWaitSecondsShutdown, Db, Bc)
	if err != nil {
		loghelper.LogError(err).Error("Ungracefully Shutdown ipld-eth-beacon-indexer!")
	} else {
		log.Info("Gracefully shutdown ipld-eth-beacon-indexer")
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
