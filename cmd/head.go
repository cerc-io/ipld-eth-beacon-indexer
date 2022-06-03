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
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vulcanize/ipld-ethcl-indexer/internal/boot"
	"github.com/vulcanize/ipld-ethcl-indexer/internal/shutdown"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
	"golang.org/x/sync/errgroup"
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

	Bc, Db, err := boot.BootApplicationWithRetry(ctx, viper.GetString("db.address"), viper.GetInt("db.port"), viper.GetString("db.name"), viper.GetString("db.username"), viper.GetString("db.password"), viper.GetString("db.driver"),
		viper.GetString("bc.address"), viper.GetInt("bc.port"), viper.GetString("bc.connectionProtocol"), viper.GetString("bc.type"), viper.GetInt("bc.bootRetryInterval"), viper.GetInt("bc.bootMaxRetry"),
		viper.GetInt("kg.increment"), "head", viper.GetBool("t.skipSync"), viper.GetInt("bc.uniqueNodeIdentifier"))
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
	if viper.GetBool("kg.processKnownGaps") {
		go func() {
			errG := new(errgroup.Group)
			errG.Go(func() error {
				errs := Bc.ProcessKnownGaps(viper.GetInt("kg.maxKnownGapsWorker"))
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
	err = shutdown.ShutdownHeadTracking(ctx, notifierCh, maxWaitSecondsShutdown, Db, Bc)
	if err != nil {
		loghelper.LogError(err).Error("Ungracefully Shutdown ipld-ethcl-indexer!")
	} else {
		log.Info("Gracefully shutdown ipld-ethcl-indexer")
	}

}

func init() {
	captureCmd.AddCommand(headCmd)
}

// Start prometheus server
func serveProm(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	srv := http.Server{
		Addr:    addr,
		Handler: mux,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			loghelper.LogError(err).WithField("endpoint", addr).Error("Error with prometheus")
		}
	}()
}
