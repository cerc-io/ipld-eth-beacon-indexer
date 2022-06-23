package beaconclient_test

import (
	"context"
	"os"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"

	//. "github.com/onsi/gomega"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/beaconclient"
)

var (
	prodConfig = Config{
		protocol:                os.Getenv("bc_protocol"),
		address:                 os.Getenv("bc_address"),
		port:                    getEnvInt(os.Getenv("bc_port")),
		dbHost:                  os.Getenv("db_host"),
		dbPort:                  getEnvInt(os.Getenv("db_port")),
		dbName:                  os.Getenv("db_name"),
		dbUser:                  os.Getenv("db_user"),
		dbPassword:              os.Getenv("db_password"),
		dbDriver:                os.Getenv("db_driver"),
		knownGapsTableIncrement: 100000000,
		bcUniqueIdentifier:      100,
		checkDb:                 false,
	}
)
var _ = Describe("Systemvalidation", Label("system"), func() {
	Describe("Run the application against a running lighthouse node", func() {
		Context("When we receive head messages", Label("system-head"), func() {
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
	//startGoRoutines := runtime.NumGoroutine()
	//pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
	ctx, cancel := context.WithCancel(context.Background())
	go bc.CaptureHead(ctx, 2, false)
	time.Sleep(5 * time.Second)
	validateMetrics(bc, expectedInserts, expectedReorgs, expectedKnownGaps, expectedKnownGapsReprocessError)

	cancel()
	time.Sleep(4 * time.Second)
	testStopSystemHeadTracking(ctx, bc)
}

// Custom stop for system testing
func testStopSystemHeadTracking(ctx context.Context, bc *beaconclient.BeaconClient) {
	bc.StopHeadTracking(ctx, false)

	time.Sleep(3 * time.Second)
	//pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
	//Expect(endNum <= startGoRoutines).To(BeTrue())
}
