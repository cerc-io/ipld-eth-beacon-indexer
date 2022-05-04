//go:build !race
// +build !race

package shutdown_test

import (
	"context"
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

var _ = Describe("Shutdown", func() {
	var (
		dbAddress              string        = "localhost"
		dbPort                 int           = 8077
		dbName                 string        = "vulcanize_testing"
		dbUsername             string        = "vdbm"
		dbPassword             string        = "password"
		dbDriver               string        = "PGX"
		bcAddress              string        = "localhost"
		bcPort                 int           = 5052
		bcConnectionProtocol   string        = "http"
		maxWaitSecondsShutdown time.Duration = time.Duration(1) * time.Second
		DB                     sql.Database
		BC                     *beaconclient.BeaconClient
		err                    error
		ctx                    context.Context
	)
	BeforeEach(func() {
		ctx = context.Background()
		BC, DB, err = boot.BootApplicationWithRetry(ctx, dbAddress, dbPort, dbName, dbUsername, dbPassword, dbDriver, bcAddress, bcPort, bcConnectionProtocol)
		Expect(err).To(BeNil())
	})

	Describe("Run Shutdown Function,", Label("integration"), func() {
		Context("When Channels are empty,", func() {
			It("Should Shutdown Successfully.", func() {
				go func() {
					log.Debug("Starting shutdown chan")
					err = shutdown.ShutdownServices(ctx, maxWaitSecondsShutdown, DB, BC)
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
					err = shutdown.ShutdownServices(ctx, maxWaitSecondsShutdown, DB, BC)
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
					log.Debug("Calling SIGHUP")
					syscall.Kill(syscall.Getpid(), syscall.SIGHUP)
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
					err = shutdown.ShutdownServices(ctx, maxWaitSecondsShutdown, DB, BC)
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
					syscall.Kill(syscall.Getpid(), syscall.SIGHUP)
				}()

				<-shutdownCh
			})
		})
	})
})
