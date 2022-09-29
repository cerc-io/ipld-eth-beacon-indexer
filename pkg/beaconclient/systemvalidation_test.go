package beaconclient_test

import (
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"

	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/beaconclient"
)

var (
	prodConfig = Config{
		protocol:                     os.Getenv("bc_protocol"),
		address:                      os.Getenv("bc_address"),
		port:                         getEnvInt(os.Getenv("bc_port")),
		dbHost:                       os.Getenv("db_host"),
		dbPort:                       getEnvInt(os.Getenv("db_port")),
		dbName:                       os.Getenv("db_name"),
		dbUser:                       os.Getenv("db_user"),
		dbPassword:                   os.Getenv("db_password"),
		dbDriver:                     os.Getenv("db_driver"),
		knownGapsTableIncrement:      100000000,
		bcUniqueIdentifier:           100,
		checkDb:                      false,
		performBeaconBlockProcessing: true,
		// As of 2022-09, generating and downloading the full BeaconState is so slow it will cause the tests to fail.
		performBeaconStateProcessing: false,
	}
)

// Note: These tests expect to communicate with a fully-synced Beacon node.

var _ = Describe("Systemvalidation", Label("system"), func() {
	level, _ := log.ParseLevel("debug")
	log.SetLevel(level)

	Describe("Run the application against a running lighthouse node", func() {
		Context("When we receive head messages", func() {
			It("We should process the messages successfully", func() {
				bc := setUpTest(prodConfig, "10000000000")
				processProdHeadBlocks(bc, 3, 0, 0, 0)
			})
		})
		Context("When we have historical and knownGaps slots to process", Label("system-batch"), func() {
			It("Should process them successfully", func() {
				bc := setUpTest(prodConfig, "10000000000")
				//known Gaps
				BeaconNodeTester.writeEventToKnownGaps(bc, 100, 101)
				BeaconNodeTester.runKnownGapsProcess(bc, 2, 2, 0, 0, 0)

				// Historical
				BeaconNodeTester.writeEventToHistoricProcess(bc, 2375703, 2375703, 10)
				BeaconNodeTester.runHistoricalProcess(bc, 2, 3, 0, 0, 0)

				time.Sleep(2 * time.Second)
				validatePopularBatchBlocks(bc)
			})
		})
	})
})

// Wrapper function to get int env variables.
func getEnvInt(envVar string) int {
	val, err := strconv.Atoi(envVar)
	if err != nil {
		return 0
	}
	return val
}

// Start head tracking and wait for the expected results.
func processProdHeadBlocks(bc *beaconclient.BeaconClient, expectedInserts, expectedReorgs, expectedKnownGaps, expectedKnownGapsReprocessError uint64) {
	go bc.CaptureHead()
	time.Sleep(1 * time.Second)
	validateMetrics(bc, expectedInserts, expectedReorgs, expectedKnownGaps, expectedKnownGapsReprocessError)
}
