package beaconclient_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/beaconclient"
)

var _ = Describe("Capturehistoric", func() {

	Describe("Run the application in historic mode", Label("unit", "behavioral"), func() {
		Context("Phase0: When we need to process a single block in the ethcl.historic_process table.", Label("now"), func() {
			It("Successfully Process the Block", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "99")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.writeEventToHistoricProcess(bc, 100, 101, 10)
				BeaconNodeTester.runBatchProcess(bc, 2, 100, 101, 0, 0)

				time.Sleep(2 * time.Second)
				validateSlot(bc, BeaconNodeTester.TestEvents["100"].HeadMessage, 3, "proposed")
				//validateSignedBeaconBlock(bc, BeaconNodeTester.TestEvents["100"].HeadMessage, "0x629ae1587895043076500f4f5dcb202a47c2fc95d5b5c548cb83bc97bd2dbfe1", "0x8d3f027beef5cbd4f8b29fc831aba67a5d74768edca529f5596f07fd207865e1", "/blocks/QHVAEQBQGQ4TKNJUGAYDGNZRGM2DOZJSGZTDMMLEG5QTIYTCMRQTKYRSGNTGCMDCGI2WINLGMM2DMNJRGYYGMMTBHEZGINJSME3DGYRZGE4WE")
				//validateBeaconState(bc, BeaconNodeTester.TestEvents["100"].HeadMessage, "/blocks/QHVAEQRQPBTDEOBWMEYDGNZZMMYDGOBWMEZWGN3CMUZDQZBQGVSDQMRZMY4GKYRXMIZDQMDDMM4WKZDFGE2TINBZMFTDEMDFMJRWIMBWME3WCNJW")

				//validateSignedBeaconBlock(bc, BeaconNodeTester.TestEvents["101"].HeadMessage, "0x629ae1587895043076500f4f5dcb202a47c2fc95d5b5c548cb83bc97bd2dbfe1", "0x8d3f027beef5cbd4f8b29fc831aba67a5d74768edca529f5596f07fd207865e1", "/blocks/QHVAEQBQGQ4TKNJUGAYDGNZRGM2DOZJSGZTDMMLEG5QTIYTCMRQTKYRSGNTGCMDCGI2WINLGMM2DMNJRGYYGMMTBHEZGINJSME3DGYRZGE4WE")
				//validateBeaconState(bc, BeaconNodeTester.TestEvents["101"].HeadMessage, "/blocks/QHVAEQRQPBTDEOBWMEYDGNZZMMYDGOBWMEZWGN3CMUZDQZBQGVSDQMRZMY4GKYRXMIZDQMDDMM4WKZDFGE2TINBZMFTDEMDFMJRWIMBWME3WCNJW")

			})
		})
	})
})

// This function will write an even to the ethcl.historic_process table
func (tbc TestBeaconNode) writeEventToHistoricProcess(bc *beaconclient.BeaconClient, startSlot, endSlot, priority int) {
	log.Debug("We are writing the necessary events to batch process")
	insertHistoricProcessingStmt := `INSERT INTO ethcl.historic_process (start_slot, end_slot, priority)
	VALUES ($1, $2, $3);`
	res, err := bc.Db.Exec(context.Background(), insertHistoricProcessingStmt, startSlot, endSlot, priority)
	Expect(err)
	rows, err := res.RowsAffected()
	if rows != 1 {
		Fail("We didnt write...")
	}
	Expect(err)
}

// Start the batch processing function, and check for the correct inserted slots.
func (tbc TestBeaconNode) runBatchProcess(bc *beaconclient.BeaconClient, maxWorkers int, startSlot uint64, endSlot uint64, expectedReorgs uint64, expectedKnownGaps uint64) {
	go bc.CaptureHistoric(maxWorkers)
	diff := endSlot - startSlot + 1

	curRetry := 0
	for atomic.LoadUint64(&bc.Metrics.SlotInserts) != diff {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			Fail(fmt.Sprintf("Too many retries have occurred. The number of inserts expects %d, the number that actually occurred, %d", atomic.LoadUint64(&bc.Metrics.SlotInserts), diff))
		}
	}

	Expect(atomic.LoadUint64(&bc.Metrics.KnownGapsInserts)).To(Equal(expectedKnownGaps))
	Expect(atomic.LoadUint64(&bc.Metrics.ReorgInserts)).To(Equal(expectedKnownGaps))
}
