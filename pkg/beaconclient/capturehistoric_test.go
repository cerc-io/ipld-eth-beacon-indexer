package beaconclient_test

import (
	"context"
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/beaconclient"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/database/sql"
)

var (
	kgCheckCheckedOutStmt = `SELECT * FROM eth_beacon.known_gaps WHERE checked_out=true `
	hpCheckCheckedOutStmt = `SELECT * FROM eth_beacon.historic_process WHERE checked_out=true `
)

var _ = Describe("Capturehistoric", func() {

	Describe("Run the application in historic mode", Label("unit", "behavioral", "historical"), func() {
		Context("Phase0 + Altairs: When we need to process a multiple blocks in a multiple entries in the eth_beacon.historic_process table.", Label("deb"), func() {
			It("Successfully Process the Blocks", func() {
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()

				bc := setUpTest(BeaconNodeTester.TestConfig, "99")
				BeaconNodeTester.writeEventToHistoricProcess(bc, 100, 101, 10)
				BeaconNodeTester.runHistoricalProcess(bc, 2, 2, 0, 0, 0)
				// Run Two seperate processes
				BeaconNodeTester.writeEventToHistoricProcess(bc, 2375703, 2375703, 10)
				BeaconNodeTester.runHistoricalProcess(bc, 2, 3, 0, 0, 0)

				validatePopularBatchBlocks(bc)

			})
		})
		Context("When the start block is greater than the endBlock", func() {
			It("Should Add two entries to the knownGaps table", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "99")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.writeEventToHistoricProcess(bc, 101, 100, 10)
				BeaconNodeTester.runHistoricalProcess(bc, 2, 0, 0, 2, 0)
			})
		})
		Context("Processing the Genesis block", Label("genesis"), func() {
			It("Should Process properly", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "100")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.writeEventToHistoricProcess(bc, 0, 0, 10)
				BeaconNodeTester.runHistoricalProcess(bc, 2, 1, 0, 0, 0)
				validateSlot(bc, BeaconNodeTester.TestEvents["0"].HeadMessage, 0, "proposed")
				validateSignedBeaconBlock(bc, BeaconNodeTester.TestEvents["0"].HeadMessage, BeaconNodeTester.TestEvents["0"].CorrectParentRoot, BeaconNodeTester.TestEvents["0"].CorrectEth1BlockHash, BeaconNodeTester.TestEvents["0"].CorrectSignedBeaconBlockMhKey)
				validateBeaconState(bc, BeaconNodeTester.TestEvents["0"].HeadMessage, BeaconNodeTester.TestEvents["0"].CorrectBeaconStateMhKey)
			})
		})
		Context("When there is a skipped slot", func() {
			It("Should process the slot properly.", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "3797055")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.writeEventToHistoricProcess(bc, 3797056, 3797056, 10)
				BeaconNodeTester.runHistoricalProcess(bc, 2, 1, 0, 0, 0)
				validateSlot(bc, BeaconNodeTester.TestEvents["3797056"].HeadMessage, 118658, "skipped")
			})
		})
	})
	Describe("Running the Application to process Known Gaps", Label("unit", "behavioral", "knownGaps"), func() {
		Context("Phase0 + Altairs: When we need to process a multiple blocks in a multiple entries in the eth_beacon.known_gaps table.", Label("leak"), func() {
			It("Successfully Process the Blocks", func() {
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				startNum := runtime.NumGoroutine()

				bc := setUpTest(BeaconNodeTester.TestConfig, "99")
				BeaconNodeTester.writeEventToKnownGaps(bc, 100, 101)
				BeaconNodeTester.runKnownGapsProcess(bc, 2, 2, 0, 0, 0)
				// Run Two seperate processes
				BeaconNodeTester.writeEventToKnownGaps(bc, 2375703, 2375703)
				BeaconNodeTester.runKnownGapsProcess(bc, 2, 3, 0, 0, 0)

				time.Sleep(2 * time.Second)
				validatePopularBatchBlocks(bc)

				bc.Db.Close()
				endNum := runtime.NumGoroutine()
				Expect(startNum).To(Equal(endNum))
			})
		})
		Context("When the start block is greater than the endBlock", func() {
			It("Should Add two entries to the knownGaps table", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "104")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.writeEventToKnownGaps(bc, 101, 100)
				BeaconNodeTester.runKnownGapsProcess(bc, 2, 2, 0, 2, 0)
			})
		})
		Context("When theres a reprocessing error", Label("reprocessingError"), func() {
			It("Should update the reprocessing error.", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "99")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				// We dont have an entry in the BeaconNodeTester for this slot
				BeaconNodeTester.writeEventToHistoricProcess(bc, 105, 105, 10)
				BeaconNodeTester.runHistoricalProcess(bc, 2, 0, 0, 1, 0)
				BeaconNodeTester.runKnownGapsProcess(bc, 2, 0, 0, 1, 1)
			})
		})
	})
	Describe("Running the application in Historic, Head, and KnownGaps mode", Label("unit", "historical", "full"), func() {
		Context("When it recieves a head, historic and known Gaps message (in order)", Label("deb"), func() {
			It("Should process them all successfully.", func() {
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()

				bc := setUpTest(BeaconNodeTester.TestConfig, "2375702")
				// Head
				BeaconNodeTester.testProcessBlock(bc, BeaconNodeTester.TestEvents["2375703"].HeadMessage, 74240, maxRetry, 1, 0, 0)

				// Historical
				BeaconNodeTester.writeEventToHistoricProcess(bc, 100, 100, 10)
				BeaconNodeTester.runHistoricalProcess(bc, 2, 2, 0, 0, 0)

				// Known Gaps
				BeaconNodeTester.writeEventToKnownGaps(bc, 101, 101)
				BeaconNodeTester.runKnownGapsProcess(bc, 2, 3, 0, 0, 0)

				time.Sleep(2 * time.Second)
				validatePopularBatchBlocks(bc)

			})
		})
		Context("When it recieves a historic, head and known Gaps message (in order)", func() {
			It("Should process them all successfully.", func() {
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()

				bc := setUpTest(BeaconNodeTester.TestConfig, "2375702")
				// Historical
				BeaconNodeTester.writeEventToHistoricProcess(bc, 100, 100, 10)
				BeaconNodeTester.runHistoricalProcess(bc, 2, 1, 0, 0, 0)

				// Head
				BeaconNodeTester.testProcessBlock(bc, BeaconNodeTester.TestEvents["2375703"].HeadMessage, 74240, maxRetry, 1, 0, 0)

				// Known Gaps
				BeaconNodeTester.writeEventToKnownGaps(bc, 101, 101)
				BeaconNodeTester.runKnownGapsProcess(bc, 2, 3, 0, 0, 0)

				time.Sleep(2 * time.Second)
				validatePopularBatchBlocks(bc)
			})
		})
		Context("When it recieves a known Gaps, historic and head  message (in order)", func() {
			It("Should process them all successfully.", func() {
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()

				bc := setUpTest(BeaconNodeTester.TestConfig, "2375702")
				// Known Gaps
				BeaconNodeTester.writeEventToKnownGaps(bc, 101, 101)
				BeaconNodeTester.runKnownGapsProcess(bc, 2, 1, 0, 0, 0)

				// Historical
				BeaconNodeTester.writeEventToHistoricProcess(bc, 100, 100, 10)
				BeaconNodeTester.runHistoricalProcess(bc, 2, 2, 0, 0, 0)

				// Head
				BeaconNodeTester.testProcessBlock(bc, BeaconNodeTester.TestEvents["2375703"].HeadMessage, 74240, maxRetry, 1, 0, 0)

				time.Sleep(2 * time.Second)
				validatePopularBatchBlocks(bc)
			})
		})
	})
})

// This function will write an even to the eth_beacon.known_gaps table
func (tbc TestBeaconNode) writeEventToKnownGaps(bc *beaconclient.BeaconClient, startSlot, endSlot int) {
	log.Debug("We are writing the necessary events to batch process")
	insertKnownGapsStmt := `INSERT INTO eth_beacon.known_gaps (start_slot, end_slot)
	VALUES ($1, $2);`
	res, err := bc.Db.Exec(context.Background(), insertKnownGapsStmt, startSlot, endSlot)
	Expect(err).ToNot(HaveOccurred())
	rows, err := res.RowsAffected()
	if rows != 1 {
		Fail("We didnt write...")
	}
	Expect(err).ToNot(HaveOccurred())
}

// This function will write an even to the eth_beacon.known_gaps table
func (tbc TestBeaconNode) writeEventToHistoricProcess(bc *beaconclient.BeaconClient, startSlot, endSlot, priority int) {
	log.Debug("We are writing the necessary events to batch process")
	insertHistoricProcessingStmt := `INSERT INTO eth_beacon.historic_process (start_slot, end_slot, priority)
	VALUES ($1, $2, $3);`
	res, err := bc.Db.Exec(context.Background(), insertHistoricProcessingStmt, startSlot, endSlot, priority)
	Expect(err).ToNot(HaveOccurred())
	rows, err := res.RowsAffected()
	if rows != 1 {
		Fail("We didnt write...")
	}
	Expect(err).ToNot(HaveOccurred())
}

// Start the CaptureHistoric function, and check for the correct inserted slots.
func (tbc TestBeaconNode) runHistoricalProcess(bc *beaconclient.BeaconClient, maxWorkers int, expectedInserts, expectedReorgs, expectedKnownGaps, expectedKnownGapsReprocessError uint64) {
	startGoRoutines := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(context.Background())
	go bc.CaptureHistoric(ctx, maxWorkers)
	validateMetrics(bc, expectedInserts, expectedReorgs, expectedKnownGaps, expectedKnownGapsReprocessError)
	cancel()
	testStopHistoricProcessing(ctx, bc, startGoRoutines)
}

// Wrapper function that processes knownGaps
func (tbc TestBeaconNode) runKnownGapsProcess(bc *beaconclient.BeaconClient, maxWorkers int, expectedInserts, expectedReorgs, expectedKnownGaps, expectedKnownGapsReprocessError uint64) {
	startGoRoutines := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(context.Background())
	go bc.ProcessKnownGaps(ctx, maxWorkers)
	validateMetrics(bc, expectedInserts, expectedReorgs, expectedKnownGaps, expectedKnownGapsReprocessError)
	cancel()
	testStopKnownGapProcessing(ctx, bc, startGoRoutines)
}

func validateMetrics(bc *beaconclient.BeaconClient, expectedInserts, expectedReorgs, expectedKnownGaps, expectedKnownGapsReprocessError uint64) {
	curRetry := 0
	value := atomic.LoadUint64(&bc.Metrics.SlotInserts)
	for value != expectedInserts {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			Fail(fmt.Sprintf("Too many retries have occurred. The number of inserts expected %d, the number that actually occurred, %d", expectedInserts, atomic.LoadUint64(&bc.Metrics.SlotInserts)))
		}
		value = atomic.LoadUint64(&bc.Metrics.SlotInserts)
	}
	curRetry = 0
	value = atomic.LoadUint64(&bc.Metrics.KnownGapsInserts)
	for value != expectedKnownGaps {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			Fail(fmt.Sprintf("Too many retries have occurred. The number of knownGaps expected %d, the number that actually occurred, %d", expectedKnownGaps, atomic.LoadUint64(&bc.Metrics.KnownGapsInserts)))
		}
		value = atomic.LoadUint64(&bc.Metrics.KnownGapsInserts)
	}
	curRetry = 0
	value = atomic.LoadUint64(&bc.Metrics.KnownGapsReprocessError)
	for value != expectedKnownGapsReprocessError {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			Fail(fmt.Sprintf("Too many retries have occurred. The number of knownGapsReprocessingErrors expected %d, the number that actually occurred, %d", expectedKnownGapsReprocessError, value))
		}
		log.Debug("&bc.Metrics.KnownGapsReprocessError: ", &bc.Metrics.KnownGapsReprocessError)
		value = atomic.LoadUint64(&bc.Metrics.KnownGapsReprocessError)
	}
	curRetry = 0
	value = atomic.LoadUint64(&bc.Metrics.ReorgInserts)
	for value != expectedReorgs {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			Fail(fmt.Sprintf("Too many retries have occurred. The number of Reorgs expected %d, the number that actually occurred, %d", expectedReorgs, atomic.LoadUint64(&bc.Metrics.ReorgInserts)))
		}
		value = atomic.LoadUint64(&bc.Metrics.ReorgInserts)
	}
}

// A wrapper function to validate a few popular blocks
func validatePopularBatchBlocks(bc *beaconclient.BeaconClient) {
	validateSlot(bc, BeaconNodeTester.TestEvents["100"].HeadMessage, 3, "proposed")
	validateSignedBeaconBlock(bc, BeaconNodeTester.TestEvents["100"].HeadMessage, BeaconNodeTester.TestEvents["100"].CorrectParentRoot, BeaconNodeTester.TestEvents["100"].CorrectEth1BlockHash, BeaconNodeTester.TestEvents["100"].CorrectSignedBeaconBlockMhKey)
	validateBeaconState(bc, BeaconNodeTester.TestEvents["100"].HeadMessage, BeaconNodeTester.TestEvents["100"].CorrectBeaconStateMhKey)

	validateSlot(bc, BeaconNodeTester.TestEvents["101"].HeadMessage, 3, "proposed")
	validateSignedBeaconBlock(bc, BeaconNodeTester.TestEvents["101"].HeadMessage, BeaconNodeTester.TestEvents["100"].HeadMessage.Block, BeaconNodeTester.TestEvents["101"].CorrectEth1BlockHash, BeaconNodeTester.TestEvents["101"].CorrectSignedBeaconBlockMhKey)
	validateBeaconState(bc, BeaconNodeTester.TestEvents["101"].HeadMessage, BeaconNodeTester.TestEvents["101"].CorrectBeaconStateMhKey)

	validateSlot(bc, BeaconNodeTester.TestEvents["2375703"].HeadMessage, 74240, "proposed")
	validateSignedBeaconBlock(bc, BeaconNodeTester.TestEvents["2375703"].HeadMessage, BeaconNodeTester.TestEvents["2375703"].CorrectParentRoot, BeaconNodeTester.TestEvents["2375703"].CorrectEth1BlockHash, BeaconNodeTester.TestEvents["2375703"].CorrectSignedBeaconBlockMhKey)
	validateBeaconState(bc, BeaconNodeTester.TestEvents["2375703"].HeadMessage, BeaconNodeTester.TestEvents["2375703"].CorrectBeaconStateMhKey)
}

// Make sure all rows have checked_out as false.
func validateAllRowsCheckedOut(db sql.Database, checkStmt string) {
	res, err := db.Exec(context.Background(), checkStmt)
	Expect(err).ToNot(HaveOccurred())
	rows, err := res.RowsAffected()
	Expect(err).ToNot(HaveOccurred())
	Expect(rows).To(Equal(int64(0)))
}

// A make shift function to stop head tracking and insure we dont have any goroutine leaks
func testStopHistoricProcessing(ctx context.Context, bc *beaconclient.BeaconClient, startGoRoutines int) {
	log.Debug("Calling the stop function for historical processing..")
	err := bc.StopHistoricProcess(ctx)
	Expect(err).ToNot(HaveOccurred())
	time.Sleep(5 * time.Second)
	validateAllRowsCheckedOut(bc.Db, hpCheckCheckedOutStmt)

	time.Sleep(3 * time.Second)
	endNum := runtime.NumGoroutine()

	//pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
	//Expect(endNum <= startGoRoutines).To(BeTrue())
	Expect(endNum).To(Equal(startGoRoutines))
}

// A make shift function to stop head tracking and insure we dont have any goroutine leaks
func testStopKnownGapProcessing(ctx context.Context, bc *beaconclient.BeaconClient, startGoRoutines int) {
	log.Debug("Calling the stop function for knownGaps processing..")
	err := bc.StopKnownGapsProcessing(ctx)
	Expect(err).ToNot(HaveOccurred())
	time.Sleep(5 * time.Second)
	validateAllRowsCheckedOut(bc.Db, kgCheckCheckedOutStmt)

	time.Sleep(3 * time.Second)
	endNum := runtime.NumGoroutine()

	//pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
	//Expect(endNum <= startGoRoutines).To(BeTrue())
	Expect(endNum).To(Equal(startGoRoutines))
}
