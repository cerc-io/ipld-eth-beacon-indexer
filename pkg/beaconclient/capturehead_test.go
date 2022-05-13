package beaconclient_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo/v2"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	st "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/r3labs/sse"
	log "github.com/sirupsen/logrus"

	. "github.com/onsi/gomega"

	"github.com/vulcanize/ipld-ethcl-indexer/pkg/beaconclient"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql/postgres"
)

type Message struct {
	HeadMessage       beaconclient.Head // The head messsage that will be streamed to the BeaconClient
	TestNotes         string            // A small explanation of the purpose this structure plays in the testing landscape.
	MimicConfig       *MimicConfig      // A configuration of parameters that you are trying to
	SignedBeaconBlock string            // The file path output of an SSZ encoded SignedBeaconBlock.
	BeaconState       string            // The file path output of an SSZ encoded BeaconState.
}

// A structure that can be utilized to mimic and existing SSZ object but change it ever so slightly.
// This is used because creating your own SSZ object is a headache.
type MimicConfig struct {
	ParentRoot string // The parent root, leave it empty if you want a to use the universal
}

var _ = Describe("Capturehead", func() {

	var (
		TestConfig              Config
		BeaconNodeTester        TestBeaconNode
		address                 string = "localhost"
		port                    int    = 8080
		protocol                string = "http"
		TestEvents              map[string]Message
		dbHost                  string = "localhost"
		dbPort                  int    = 8077
		dbName                  string = "vulcanize_testing"
		dbUser                  string = "vdbm"
		dbPassword              string = "password"
		dbDriver                string = "pgx"
		dummyParentRoot         string = "46f98c08b54a71dfda4d56e29ec3952b8300cd8d6b67a9b6c562ae96a7a25a42"
		knownGapsTableIncrement int    = 100000
		maxRetry                int    = 30
	)

	BeforeEach(func() {
		TestEvents = map[string]Message{
			"100-dummy": {
				HeadMessage: beaconclient.Head{
					Slot:                      "100",
					Block:                     "04955400371347e26f61d7a4bbda5b23fa0b25d5fc465160f2a92d52a63b919b",
					State:                     "36d5c9a129979b4502bd9a06e57a742810ecbc3fa55a0361c0723c92c1782bfa",
					CurrentDutyDependentRoot:  "",
					PreviousDutyDependentRoot: "",
					EpochTransition:           false,
					ExecutionOptimistic:       false,
				},
				TestNotes:         "A block that is supposed to replicate slot 100, but contains some dummy test information.",
				MimicConfig:       &MimicConfig{},
				SignedBeaconBlock: filepath.Join("ssz-data", "100", "signed-beacon-block.ssz"),
				BeaconState:       filepath.Join("ssz-data", "100", "beacon-state.ssz"),
			},
			"100-dummy-2": {
				HeadMessage: beaconclient.Head{
					Slot:                      "100",
					Block:                     "04955400371347e26f61d7a4bbda5b23fa0b25d5fc465160f2a9aaaaaaaaaaaa",
					State:                     "36d5c9a129979b4502bd9a06e57a742810ecbc3fa55a0361c072bbbbbbbbbbbb",
					CurrentDutyDependentRoot:  "",
					PreviousDutyDependentRoot: "",
					EpochTransition:           false,
					ExecutionOptimistic:       false,
				},
				TestNotes:         "A block that is supposed to replicate slot 100, but contains some dummy test information.",
				MimicConfig:       &MimicConfig{},
				SignedBeaconBlock: filepath.Join("ssz-data", "100", "signed-beacon-block.ssz"),
				BeaconState:       filepath.Join("ssz-data", "100", "beacon-state.ssz"),
			},
			"100": {
				HeadMessage: beaconclient.Head{
					Slot:                      "100",
					Block:                     "0x582187e97f7520bb69eea014c3834c964c45259372a0eaaea3f032013797996b",
					State:                     "0xf286a0379c0386a3c7be28d05d829f8eb7b280cc9ede15449af20ebcd06a7a56",
					CurrentDutyDependentRoot:  "",
					PreviousDutyDependentRoot: "",
					EpochTransition:           false,
					ExecutionOptimistic:       false,
				},
				TestNotes:         "An easy to process Phase 0 block",
				SignedBeaconBlock: filepath.Join("ssz-data", "100", "signed-beacon-block.ssz"),
				BeaconState:       filepath.Join("ssz-data", "100", "beacon-state.ssz"),
			},
			"2375703-dummy": {
				HeadMessage: beaconclient.Head{
					Slot:                      "2375703",
					Block:                     "c9fb337b62e2a0dae4f27ab49913132570f7f2cab3f23ad99f4d07508a8e648e",
					State:                     "0299a145bcda2c8f5e7d2e068ee101861edbee2ec1db2d5e1d850b0d265aef5f",
					CurrentDutyDependentRoot:  "",
					PreviousDutyDependentRoot: "",
					EpochTransition:           false,
					ExecutionOptimistic:       false,
				},
				TestNotes:         "This is a dummy message that is used for reorgs",
				MimicConfig:       &MimicConfig{},
				SignedBeaconBlock: filepath.Join("ssz-data", "2375703", "signed-beacon-block.ssz"),
				BeaconState:       filepath.Join("ssz-data", "2375703", "beacon-state.ssz"),
			},
			"2375703-dummy-2": {
				HeadMessage: beaconclient.Head{
					Slot:                      "2375703",
					Block:                     "c9fb337b62e2a0dae4f27ab49913132570f7f2cab3f23ad99f4d07508aaaaaaa",
					State:                     "0299a145bcda2c8f5e7d2e068ee101861edbee2ec1db2d5e1d850b0d2bbbbbbb",
					CurrentDutyDependentRoot:  "",
					PreviousDutyDependentRoot: "",
					EpochTransition:           false,
					ExecutionOptimistic:       false,
				},
				TestNotes:         "This is a dummy message that is used for reorgs",
				MimicConfig:       &MimicConfig{},
				SignedBeaconBlock: filepath.Join("ssz-data", "2375703", "signed-beacon-block.ssz"),
				BeaconState:       filepath.Join("ssz-data", "2375703", "beacon-state.ssz"),
			},
			"2375703": {
				HeadMessage: beaconclient.Head{
					Slot:                     "2375703",
					Block:                    "0x4392372c5f6e39499e31bf924388b5815639103149f0f54f8a453773b1802301",
					State:                    "0xb6215b560273af63ec7e011572b60ec1ca0b0232f8ff44fcd4ed55c7526e964e",
					CurrentDutyDependentRoot: "", PreviousDutyDependentRoot: "", EpochTransition: false, ExecutionOptimistic: false},
				TestNotes:         "An easy to process Altair Block",
				SignedBeaconBlock: filepath.Join("ssz-data", "2375703", "signed-beacon-block.ssz"),
				BeaconState:       filepath.Join("ssz-data", "2375703", "beacon-state.ssz"),
			},
		}
		TestConfig = Config{
			protocol:                protocol,
			address:                 address,
			port:                    port,
			dummyParentRoot:         dummyParentRoot,
			dbHost:                  dbHost,
			dbPort:                  dbPort,
			dbName:                  dbName,
			dbUser:                  dbUser,
			dbPassword:              dbPassword,
			dbDriver:                dbDriver,
			knownGapsTableIncrement: knownGapsTableIncrement,
		}

		BeaconNodeTester = TestBeaconNode{
			TestEvents: TestEvents,
			TestConfig: TestConfig,
		}
	})

	// We might also want to add an integration test that will actually process a single event, then end.
	// This will help us know that our models match that actual data being served from the beacon node.

	Describe("Receiving New Head SSE messages", Label("unit", "behavioral"), func() {
		Context("Correctly formatted Phase0 Block", func() {
			It("Should turn it into a struct successfully.", func() {
				BeaconNodeTester.testProcessBlock(BeaconNodeTester.TestEvents["100"].HeadMessage, 3, maxRetry)

			})
		})
		Context("Correctly formatted Altair Block", func() {
			It("Should turn it into a struct successfully.", func() {
				BeaconNodeTester.testProcessBlock(BeaconNodeTester.TestEvents["2375703"].HeadMessage, 74240, maxRetry)
			})
		})
		//Context("A single incorrectly formatted head message", func() {
		//	It("Should create an error, maybe also add the projected slot to the knownGaps table......")
		// If it can unmarshal the head add it to knownGaps
		//})
		//Context("An incorrectly formatted message sandwiched between correctly formatted messages", func() {
		//	It("Should create an error, maybe also add the projected slot to the knownGaps table......")
		//})
		//	Context("When there is a skipped slot", func() {
		//		It("Should indicate that the slot was skipped")
		//	})
		//	Context("When the slot is not properly served", func() {
		//		It("Should return an error, and add the slot to the knownGaps table.")
		//	})
		//})
		//	Context("With gaps in between head slots", func() {
		//		It("Should add the slots in between to the knownGaps table")
		//	})
		//	Context("With the previousBlockHash not matching the parentBlockHash", func() {
		//		It("Should recognize the reorg and add the previous slot to knownGaps table.")
		//	})
		//	Context("Out of order", func() {
		//		It("Not sure what it should do....")
		//	})
	})

	Describe("ReOrg Scenario", Label("unit", "behavioral"), func() {
		Context("Altair: Multiple head messages for the same slot.", func() {
			It("The previous block should be marked as 'forked', the new block should be the only one marked as 'proposed'.", func() {
				BeaconNodeTester.testMultipleHead(TestEvents["2375703-dummy"].HeadMessage, TestEvents["2375703"].HeadMessage, 74240, maxRetry)
			})
		})
		Context("Phase0: Multiple head messages for the same slot.", func() {
			It("The previous block should be marked as 'forked', the new block should be the only one marked as 'proposed'.", func() {
				BeaconNodeTester.testMultipleHead(TestEvents["100-dummy"].HeadMessage, TestEvents["100"].HeadMessage, 3, maxRetry)
			})
		})
		Context("Phase 0: Multiple reorgs have occurred on this slot", Label("new"), func() {
			It("The previous blocks should be marked as 'forked', the new block should be the only one marked as 'proposed'.", func() {
				BeaconNodeTester.testMultipleReorgs(TestEvents["100-dummy"].HeadMessage, TestEvents["100-dummy-2"].HeadMessage, TestEvents["100"].HeadMessage, 3, maxRetry)
			})
		})
		Context("Altair: Multiple reorgs have occurred on this slot", Label("new"), func() {
			It("The previous blocks should be marked as 'forked', the new block should be the only one marked as 'proposed'.", func() {
				BeaconNodeTester.testMultipleReorgs(TestEvents["2375703-dummy"].HeadMessage, TestEvents["2375703-dummy-2"].HeadMessage, TestEvents["2375703"].HeadMessage, 74240, maxRetry)
			})
		})
		//Context("Reorg slot in not already in the DB", func() {
		//	It("Should simply have the correct slot in the DB.")
		// Add to knowngaps
		//})

	})
})

type Config struct {
	protocol                string
	address                 string
	port                    int
	dummyParentRoot         string
	dbHost                  string
	dbPort                  int
	dbName                  string
	dbUser                  string
	dbPassword              string
	dbDriver                string
	knownGapsTableIncrement int
}

//////////////////////////////////////////////////////
// Helper functions
//////////////////////////////////////////////////////

// Must run before each test. We can't use the beforeEach because of the way
// Gingko treats race conditions.
func setUpTest(config Config) *beaconclient.BeaconClient {
	bc := *beaconclient.CreateBeaconClient(context.Background(), config.protocol, config.address, config.port)
	db, err := postgres.SetupPostgresDb(config.dbHost, config.dbPort, config.dbName, config.dbUser, config.dbPassword, config.dbDriver)
	Expect(err).ToNot(HaveOccurred())

	// Drop all records from the DB.
	clearEthclDbTables(db)
	bc.Db = db

	return &bc
}

// A helper function to validate the expected output.
func validateSlot(bc *beaconclient.BeaconClient, headMessage *beaconclient.Head, correctEpoch int, correctStatus string) {
	epoch, dbSlot, blockRoot, stateRoot, status := queryDbSlotAndBlock(bc.Db, headMessage.Slot, headMessage.Block)
	baseSlot, err := strconv.Atoi(headMessage.Slot)
	Expect(err).ToNot(HaveOccurred())
	Expect(dbSlot).To(Equal(baseSlot))
	Expect(epoch).To(Equal(correctEpoch))
	Expect(blockRoot).To(Equal(headMessage.Block))
	Expect(stateRoot).To(Equal(headMessage.State))
	Expect(status).To(Equal(correctStatus))
}

// Wrapper function to send a head message to the beaconclient
func sendHeadMessage(bc *beaconclient.BeaconClient, head beaconclient.Head, maxRetry int) {

	data, err := json.Marshal(head)
	Expect(err).ToNot(HaveOccurred())

	startInserts := atomic.LoadUint64(&bc.Metrics.HeadTrackingInserts)
	bc.HeadTracking.MessagesCh <- &sse.Event{
		ID:    []byte{},
		Data:  data,
		Event: []byte{},
		Retry: []byte{},
	}
	curRetry := 0
	for atomic.LoadUint64(&bc.Metrics.HeadTrackingInserts) != startInserts+1 {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			Fail(" Too many retries have occured.")
		}
	}
}

// A helper function to query the ethcl.slots table based on the slot and block_root
func queryDbSlotAndBlock(db sql.Database, querySlot string, queryBlockRoot string) (int, int, string, string, string) {
	sqlStatement := `SELECT epoch, slot, block_root, state_root, status FROM ethcl.slots WHERE slot=$1 AND block_root=$2;`
	var epoch, slot int
	var blockRoot, stateRoot, status string
	row := db.QueryRow(context.Background(), sqlStatement, querySlot, queryBlockRoot)
	err := row.Scan(&epoch, &slot, &blockRoot, &stateRoot, &status)
	Expect(err).ToNot(HaveOccurred())
	return epoch, slot, blockRoot, stateRoot, status
}

// A function that will remove all entries from the ethcl tables for you.
func clearEthclDbTables(db sql.Database) {
	deleteQueries := []string{"DELETE FROM ethcl.slots;", "DELETE FROM ethcl.signed_beacon_block;", "DELETE FROM ethcl.beacon_state;", "DELETE FROM ethcl.known_gaps;"}
	for _, queries := range deleteQueries {
		_, err := db.Exec(context.Background(), queries)
		Expect(err).ToNot(HaveOccurred())
	}

}

// An object that is used to aggregate test functions. Test functions are needed because we need to
// run the same tests on multiple blocks for multiple forks. So they save us time.
type TestBeaconNode struct {
	TestEvents map[string]Message
	TestConfig Config
}

// Create a new new mock for the beacon node.
func (tbc TestBeaconNode) SetupBeaconNodeMock(TestEvents map[string]Message, protocol string, address string, port int, dummyParentRoot string) {
	httpmock.Activate()
	stateUrl := `=~^` + protocol + "://" + address + ":" + strconv.Itoa(port) + beaconclient.BcStateQueryEndpoint + `([^/]+)\z`
	httpmock.RegisterResponder("GET", stateUrl,
		func(req *http.Request) (*http.Response, error) {
			// Get ID from request
			id := httpmock.MustGetSubmatch(req, 1)
			dat, err := tbc.provideSsz(id, "state", dummyParentRoot)
			if err != nil {
				Expect(err).NotTo(HaveOccurred())
				return httpmock.NewStringResponse(404, fmt.Sprintf("Unable to find file for %s", id)), err
			}
			return httpmock.NewBytesResponse(200, dat), nil
		},
	)

	blockUrl := `=~^` + protocol + "://" + address + ":" + strconv.Itoa(port) + beaconclient.BcBlockQueryEndpoint + `([^/]+)\z`
	httpmock.RegisterResponder("GET", blockUrl,
		func(req *http.Request) (*http.Response, error) {
			// Get ID from request
			id := httpmock.MustGetSubmatch(req, 1)
			dat, err := tbc.provideSsz(id, "block", dummyParentRoot)
			if err != nil {
				Expect(err).NotTo(HaveOccurred())
				return httpmock.NewStringResponse(404, fmt.Sprintf("Unable to find file for %s", id)), err
			}
			return httpmock.NewBytesResponse(200, dat), nil
		},
	)
}

// A function to mimic querying the state from the beacon node. We simply get the SSZ file are return it.
func (tbc TestBeaconNode) provideSsz(slotIdentifier string, sszIdentifier string, dummyParentRoot string) ([]byte, error) {
	var slotFile string
	var Message Message

	for _, val := range tbc.TestEvents {
		if sszIdentifier == "state" {
			if val.HeadMessage.Slot == slotIdentifier || val.HeadMessage.State == slotIdentifier {
				slotFile = val.BeaconState
				Message = val
			}
		} else if sszIdentifier == "block" {
			if val.HeadMessage.Slot == slotIdentifier || val.HeadMessage.Block == slotIdentifier {
				slotFile = val.SignedBeaconBlock
				Message = val
			}
		}
	}

	if Message.MimicConfig != nil {
		log.Info("We are going to create a custom SSZ object for testing purposes.")
		if sszIdentifier == "block" {
			dat, err := os.ReadFile(slotFile)
			if err != nil {
				return nil, fmt.Errorf("Can't find the slot file, %s", slotFile)
			}

			block := &st.SignedBeaconBlock{}
			err = block.UnmarshalSSZ(dat)
			if err != nil {
				log.Error("Error unmarshalling: ", err)
			}

			slot, err := strconv.ParseUint(Message.HeadMessage.Slot, 10, 64)
			Expect(err).ToNot(HaveOccurred())
			block.Block.Slot = types.Slot(slot)

			block.Block.StateRoot, err = hex.DecodeString(Message.HeadMessage.State)
			Expect(err).ToNot(HaveOccurred())

			if Message.MimicConfig.ParentRoot == "" {
				block.Block.ParentRoot, err = hex.DecodeString(dummyParentRoot)
				Expect(err).ToNot(HaveOccurred())
			} else {
				block.Block.ParentRoot, err = hex.DecodeString(Message.MimicConfig.ParentRoot)
				Expect(err).ToNot(HaveOccurred())
			}
			return block.MarshalSSZ()
		}
		if sszIdentifier == "state" {
			dat, err := os.ReadFile(slotFile)
			if err != nil {
				return nil, fmt.Errorf("Can't find the slot file, %s", slotFile)
			}
			state := st.BeaconState{}
			err = state.UnmarshalSSZ(dat)
			Expect(err)
			slot, err := strconv.ParseUint(Message.HeadMessage.Slot, 10, 64)
			Expect(err).ToNot(HaveOccurred())
			state.Slot = types.Slot(slot)
			return state.MarshalSSZ()
		}
	}

	if slotFile == "" {
		return nil, fmt.Errorf("We couldn't find the slot file for %s", slotIdentifier)
	}

	dat, err := os.ReadFile(slotFile)
	if err != nil {
		return nil, fmt.Errorf("Can't find the slot file, %s", slotFile)
	}
	return dat, nil
}

// Helper function to test three reorg messages. There are going to be many functions like this,
// Because we need to test the same logic for multiple phases.
func (tbc TestBeaconNode) testMultipleReorgs(firstHead beaconclient.Head, secondHead beaconclient.Head, thirdHead beaconclient.Head, epoch int, maxRetry int) {
	bc := setUpTest(tbc.TestConfig)
	tbc.SetupBeaconNodeMock(tbc.TestEvents, tbc.TestConfig.protocol, tbc.TestConfig.address, tbc.TestConfig.port, tbc.TestConfig.dummyParentRoot)
	defer httpmock.DeactivateAndReset()

	go bc.CaptureHead(tbc.TestConfig.knownGapsTableIncrement)
	time.Sleep(1 * time.Second)

	log.Info("Sending Phase0 Messages to BeaconClient")
	sendHeadMessage(bc, firstHead, maxRetry)
	sendHeadMessage(bc, secondHead, maxRetry)
	sendHeadMessage(bc, thirdHead, maxRetry)

	curRetry := 0
	for atomic.LoadUint64(&bc.Metrics.HeadTrackingReorgs) != 2 {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			Fail(" Too many retries have occured.")
		}
	}

	log.Info("Checking Phase0 to make sure the fork was marked properly.")
	validateSlot(bc, &firstHead, epoch, "forked")
	validateSlot(bc, &secondHead, epoch, "forked")
	validateSlot(bc, &thirdHead, epoch, "proposed")

	log.Info("Send the reorg message.")

	data, err := json.Marshal(&beaconclient.ChainReorg{
		Slot:                firstHead.Slot,
		Depth:               "1",
		OldHeadBlock:        thirdHead.Block,
		NewHeadBlock:        secondHead.Block,
		OldHeadState:        thirdHead.State,
		NewHeadState:        secondHead.State,
		Epoch:               strconv.Itoa(epoch),
		ExecutionOptimistic: false,
	})
	Expect(err).ToNot(HaveOccurred())
	bc.ReOrgTracking.MessagesCh <- &sse.Event{
		Data: data,
	}

	curRetry = 0
	for atomic.LoadUint64(&bc.Metrics.HeadTrackingReorgs) != 3 {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			Fail(" Too many retries have occured.")
		}
	}

	log.Info("Make sure the forks were properly updated!")

	validateSlot(bc, &firstHead, epoch, "forked")
	validateSlot(bc, &secondHead, epoch, "proposed")
	validateSlot(bc, &thirdHead, epoch, "forked")

}

// A test to validate a single block was processed correctly
func (tbc TestBeaconNode) testProcessBlock(head beaconclient.Head, epoch int, maxRetry int) {
	bc := setUpTest(tbc.TestConfig)
	tbc.SetupBeaconNodeMock(tbc.TestEvents, tbc.TestConfig.protocol, tbc.TestConfig.address, tbc.TestConfig.port, tbc.TestConfig.dummyParentRoot)
	defer httpmock.DeactivateAndReset()

	go bc.CaptureHead(tbc.TestConfig.knownGapsTableIncrement)
	time.Sleep(1 * time.Second)
	sendHeadMessage(bc, head, maxRetry)
	validateSlot(bc, &head, epoch, "proposed")
}

// A test that ensures that if two HeadMessages occur for a single slot they are marked
// as proposed and forked correctly.
func (tbc TestBeaconNode) testMultipleHead(firstHead beaconclient.Head, secondHead beaconclient.Head, epoch int, maxRetry int) {
	bc := setUpTest(tbc.TestConfig)
	tbc.SetupBeaconNodeMock(tbc.TestEvents, tbc.TestConfig.protocol, tbc.TestConfig.address, tbc.TestConfig.port, tbc.TestConfig.dummyParentRoot)
	defer httpmock.DeactivateAndReset()

	go bc.CaptureHead(tbc.TestConfig.knownGapsTableIncrement)
	time.Sleep(1 * time.Second)

	sendHeadMessage(bc, firstHead, maxRetry)
	sendHeadMessage(bc, secondHead, maxRetry)

	curRetry := 0
	for atomic.LoadUint64(&bc.Metrics.HeadTrackingReorgs) != 1 {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			Fail(" Too many retries have occured.")
		}
	}

	log.Info("Checking Altair to make sure the fork was marked properly.")
	validateSlot(bc, &firstHead, epoch, "forked")
	validateSlot(bc, &secondHead, epoch, "proposed")
}
