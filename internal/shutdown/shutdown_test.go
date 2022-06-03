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
//go:build !race
// +build !race

package shutdown_test

import (
	"context"
	"os"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/r3labs/sse"
	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/internal/boot"
	"github.com/vulcanize/ipld-ethcl-indexer/internal/shutdown"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/beaconclient"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/gracefulshutdown"
)

var (
	dbAddress              string        = "localhost"
	dbPort                 int           = 8076
	dbName                 string        = "vulcanize_testing"
	dbUsername             string        = "vdbm"
	dbPassword             string        = "password"
	dbDriver               string        = "PGX"
	bcAddress              string        = "localhost"
	bcPort                 int           = 5052
	bcConnectionProtocol   string        = "http"
	bcType                 string        = "lighthouse"
	bcBootRetryInterval    int           = 1
	bcBootMaxRetry         int           = 5
	bcKgTableIncrement     int           = 10
	bcUniqueIdentifier     int           = 100
	maxWaitSecondsShutdown time.Duration = time.Duration(1) * time.Second
	DB                     sql.Database
	BC                     *beaconclient.BeaconClient
	err                    error
	ctx                    context.Context
	notifierCh             chan os.Signal
)

var _ = Describe("Shutdown", func() {
	BeforeEach(func() {
		ctx = context.Background()
		BC, DB, err = boot.BootApplicationWithRetry(ctx, dbAddress, dbPort, dbName, dbUsername, dbPassword, dbDriver, bcAddress,
			bcPort, bcConnectionProtocol, bcType, bcBootRetryInterval, bcBootMaxRetry, bcKgTableIncrement, "head", true, bcUniqueIdentifier)
		notifierCh = make(chan os.Signal, 1)
		Expect(err).To(BeNil())
	})

	Describe("Run Shutdown Function for head tracking,", Label("integration"), func() {
		Context("When Channels are empty,", func() {
			It("Should Shutdown Successfully.", func() {
				go func() {
					log.Debug("Starting shutdown chan")
					err = shutdown.ShutdownHeadTracking(ctx, notifierCh, maxWaitSecondsShutdown, DB, BC)
					log.Debug("We have completed the shutdown...")
					Expect(err).ToNot(HaveOccurred())
				}()
			})
		})
		Context("When the Channels are not empty,", func() {
			It("Should try to clear them and shutdown gracefully.", func() {
				shutdownCh := make(chan bool)
				//log.SetLevel(log.DebugLevel)
				go func() {
					log.Debug("Starting shutdown chan")
					err = shutdown.ShutdownHeadTracking(ctx, notifierCh, maxWaitSecondsShutdown, DB, BC)
					log.Debug("We have completed the shutdown...")
					Expect(err).ToNot(HaveOccurred())
					shutdownCh <- true
				}()

				messageAddCh := make(chan bool)
				go func() {
					log.Debug("Adding messages to Channels")
					BC.HeadTracking.MessagesCh <- &sse.Event{}
					//BC.FinalizationTracking.MessagesCh <- &sse.Event{}
					BC.ReOrgTracking.MessagesCh <- &sse.Event{}
					log.Debug("Message adding complete")
					messageAddCh <- true
				}()

				go func() {
					<-messageAddCh
					log.Debug("Calling SIGTERM")
					notifierCh <- syscall.SIGTERM
					log.Debug("Reading messages from channel")
					<-BC.HeadTracking.MessagesCh
					//<-BC.FinalizationTracking.MessagesCh
					<-BC.ReOrgTracking.MessagesCh
				}()
				<-shutdownCh

			})
			It("Should try to clear them, if it can't, shutdown within a given time frame.", func() {
				shutdownCh := make(chan bool)
				//log.SetLevel(log.DebugLevel)
				go func() {
					log.Debug("Starting shutdown chan")
					err = shutdown.ShutdownHeadTracking(ctx, notifierCh, maxWaitSecondsShutdown, DB, BC)
					log.Debug("We have completed the shutdown...")
					Expect(err).To(MatchError(gracefulshutdown.TimeoutErr(maxWaitSecondsShutdown.String())))
					shutdownCh <- true
				}()

				go func() {
					log.Debug("Adding messages to Channels")
					BC.HeadTracking.MessagesCh <- &sse.Event{}
					//BC.FinalizationTracking.MessagesCh <- &sse.Event{}
					BC.ReOrgTracking.MessagesCh <- &sse.Event{}
					log.Debug("Message adding complete")
					log.Debug("Calling SIGHUP")
					notifierCh <- syscall.SIGTERM
				}()

				<-shutdownCh
			})
		})
	})
})
