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
	HeadMessage       beaconclient.Head       // The head messsage that will be streamed to the BeaconClient
	ReorgMessage      beaconclient.ChainReorg // The reorg messsage that will be streamed to the BeaconClient
	TestNotes         string                  // A small explanation of the purpose this structure plays in the testing landscape.
	MimicConfig       *MimicConfig            // A configuration of parameters that you are trying to
	SignedBeaconBlock string                  // The file path output of an SSZ encoded SignedBeaconBlock.
	BeaconState       string                  // The file path output of an SSZ encoded BeaconState.
}

// A structure that can be utilized to mimic and existing SSZ object but change it ever so slightly.
// This is used because creating your own SSZ object is a headache.
type MimicConfig struct {
	ParentRoot string // The parent root, leave it empty if you want a to use the universal
}

var _ = Describe("Capturehead", func() {

	var (
		address         string = "localhost"
		port            int    = 8080
		protocol        string = "http"
		TestEvents      map[string]*Message
		dbHost          string = "localhost"
		dbPort          int    = 8077
		dbName          string = "vulcanize_testing"
		dbUser          string = "vdbm"
		dbPassword      string = "password"
		dbDriver        string = "pgx"
		dummyParentRoot string = "46f98c08b54a71dfda4d56e29ec3952b8300cd8d6b67a9b6c562ae96a7a25a42"
	)

	BeforeEach(func() {
		TestEvents = map[string]*Message{
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
				TestNotes:         "An easy to process Phase 0 block",
				MimicConfig:       &MimicConfig{},
				SignedBeaconBlock: filepath.Join("data", "100", "signed-beacon-block.ssz"),
				BeaconState:       filepath.Join("data", "100", "beacon-state.ssz"),
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
				SignedBeaconBlock: filepath.Join("data", "100", "signed-beacon-block.ssz"),
				BeaconState:       filepath.Join("data", "100", "beacon-state.ssz"),
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
				SignedBeaconBlock: filepath.Join("data", "2375703", "signed-beacon-block.ssz"),
				BeaconState:       filepath.Join("data", "2375703", "beacon-state.ssz"),
			},
			"2375703": {
				HeadMessage: beaconclient.Head{
					Slot:                     "2375703",
					Block:                    "0x4392372c5f6e39499e31bf924388b5815639103149f0f54f8a453773b1802301",
					State:                    "0xb6215b560273af63ec7e011572b60ec1ca0b0232f8ff44fcd4ed55c7526e964e",
					CurrentDutyDependentRoot: "", PreviousDutyDependentRoot: "", EpochTransition: false, ExecutionOptimistic: false},
				ReorgMessage:      beaconclient.ChainReorg{},
				TestNotes:         "An easy to process Altair Block",
				SignedBeaconBlock: filepath.Join("data", "2375703", "signed-beacon-block.ssz"),
				BeaconState:       filepath.Join("data", "2375703", "beacon-state.ssz"),
			},
		}
	})

	// We might also want to add an integration test that will actually process a single event, then end.
	// This will help us know that our models match that actual data being served from the beacon node.

	Describe("Receiving New Head SSE messages", Label("unit"), func() {
		Context("Correctly formatted Phase0 Block", func() {
			It("Should turn it into a struct successfully.", func() {

				BC := setUpTest(TestEvents, protocol, address, port, dummyParentRoot, dbHost, dbPort, dbName, dbUser, dbPassword, dbDriver)
				SetupBeaconNodeMock(TestEvents, protocol, address, port, dummyParentRoot)
				defer httpmock.Deactivate()

				go BC.CaptureHead()
				sendHeadMessage(BC, TestEvents["100"].HeadMessage)

				epoch, slot, blockRoot, stateRoot, status := queryDbSlotAndBlock(BC.Db, TestEvents["100"].HeadMessage.Slot, TestEvents["100"].HeadMessage.Block)
				Expect(slot).To(Equal(100))
				Expect(epoch).To(Equal(3))
				Expect(blockRoot).To(Equal(TestEvents["100"].HeadMessage.Block))
				Expect(stateRoot).To(Equal(TestEvents["100"].HeadMessage.State))
				Expect(status).To(Equal("proposed"))

			})
		})
		Context("Correctly formatted Altair Block", func() {
			It("Should turn it into a struct successfully.", func() {

				BC := setUpTest(TestEvents, protocol, address, port, dummyParentRoot, dbHost, dbPort, dbName, dbUser, dbPassword, dbDriver)

				SetupBeaconNodeMock(TestEvents, protocol, address, port, dummyParentRoot)
				defer httpmock.Deactivate()

				go BC.CaptureHead()
				sendHeadMessage(BC, TestEvents["2375703"].HeadMessage)

				epoch, slot, blockRoot, stateRoot, status := queryDbSlotAndBlock(BC.Db, TestEvents["2375703"].HeadMessage.Slot, TestEvents["2375703"].HeadMessage.Block)
				Expect(slot).To(Equal(2375703))
				Expect(epoch).To(Equal(74240))
				Expect(blockRoot).To(Equal(TestEvents["2375703"].HeadMessage.Block))
				Expect(stateRoot).To(Equal(TestEvents["2375703"].HeadMessage.State))
				Expect(status).To(Equal("proposed"))

			})
		})
		//Context("A single incorrectly formatted head message", func() {
		//	It("Should create an error, maybe also add the projected slot to the knownGaps table......")
		//})
		//Context("An incorrectly formatted message sandwiched between correctly formatted messages", func() {
		//	It("Should create an error, maybe also add the projected slot to the knownGaps table......")
		//})
	})

	Describe("Receiving New Reorg SSE messages", Label("dry"), func() {
		Context("Altair: Reorg slot is already in the DB", func() {
			It("The previous block should be marked as 'forked', the new block should be the only one marked as 'proposed'.", func() {

				BC := setUpTest(TestEvents, protocol, address, port, dummyParentRoot, dbHost, dbPort, dbName, dbUser, dbPassword, dbDriver)
				SetupBeaconNodeMock(TestEvents, protocol, address, port, dummyParentRoot)
				defer httpmock.Deactivate()

				go BC.CaptureHead()
				log.Info("Sending Altair Messages to BeaconClient")
				sendHeadMessage(BC, TestEvents["2375703-dummy"].HeadMessage)
				sendHeadMessage(BC, TestEvents["2375703"].HeadMessage)

				for atomic.LoadUint64(&BC.Metrics.HeadTrackingReorgs) != 1 {
					time.Sleep(1 * time.Second)
				}

				log.Info("Checking Altair to make sure the fork was marked properly.")
				epoch, slot, blockRoot, stateRoot, status := queryDbSlotAndBlock(BC.Db, TestEvents["2375703-dummy"].HeadMessage.Slot, TestEvents["2375703-dummy"].HeadMessage.Block)
				Expect(slot).To(Equal(2375703))
				Expect(epoch).To(Equal(74240))
				Expect(blockRoot).To(Equal(TestEvents["2375703-dummy"].HeadMessage.Block))
				Expect(stateRoot).To(Equal(TestEvents["2375703-dummy"].HeadMessage.State))
				Expect(status).To(Equal("forked"))

				epoch, slot, blockRoot, stateRoot, status = queryDbSlotAndBlock(BC.Db, TestEvents["2375703"].HeadMessage.Slot, TestEvents["2375703"].HeadMessage.Block)
				Expect(slot).To(Equal(2375703))
				Expect(epoch).To(Equal(74240))
				Expect(blockRoot).To(Equal(TestEvents["2375703"].HeadMessage.Block))
				Expect(stateRoot).To(Equal(TestEvents["2375703"].HeadMessage.State))
				Expect(status).To(Equal("proposed"))
			})
		})
		Context("Phase0: Reorg slot is already in the DB", func() {
			It("The previous block should be marked as 'forked', the new block should be the only one marked as 'proposed'.", func() {

				BC := setUpTest(TestEvents, protocol, address, port, dummyParentRoot, dbHost, dbPort, dbName, dbUser, dbPassword, dbDriver)
				SetupBeaconNodeMock(TestEvents, protocol, address, port, dummyParentRoot)
				defer httpmock.Deactivate()

				go BC.CaptureHead()
				log.Info("Sending Phase0 Messages to BeaconClient")
				sendHeadMessage(BC, TestEvents["100-dummy"].HeadMessage)
				sendHeadMessage(BC, TestEvents["100"].HeadMessage)

				for atomic.LoadUint64(&BC.Metrics.HeadTrackingReorgs) != 1 {
					time.Sleep(1 * time.Second)
				}

				log.Info("Checking Phase0 to make sure the fork was marked properly.")
				epoch, slot, blockRoot, stateRoot, status := queryDbSlotAndBlock(BC.Db, TestEvents["100-dummy"].HeadMessage.Slot, TestEvents["100-dummy"].HeadMessage.Block)
				Expect(slot).To(Equal(100))
				Expect(epoch).To(Equal(3))
				Expect(blockRoot).To(Equal(TestEvents["100-dummy"].HeadMessage.Block))
				Expect(stateRoot).To(Equal(TestEvents["100-dummy"].HeadMessage.State))
				Expect(status).To(Equal("forked"))

				epoch, slot, blockRoot, stateRoot, status = queryDbSlotAndBlock(BC.Db, TestEvents["100"].HeadMessage.Slot, TestEvents["100"].HeadMessage.Block)
				Expect(slot).To(Equal(100))
				Expect(epoch).To(Equal(3))
				Expect(blockRoot).To(Equal(TestEvents["100"].HeadMessage.Block))
				Expect(stateRoot).To(Equal(TestEvents["100"].HeadMessage.State))
				Expect(status).To(Equal("proposed"))
			})
		})
		//Context("Multiple reorgs have occurred on this slot", func() {
		//	It("The previous blocks should be marked as 'forked', the new block should be the only one marked as 'proposed'.")
		//})
		//Context("Reorg slot in not already in the DB", func() {
		//	It("Should simply have the correct slot in the DB.")
		//})

	})

	//Describe("Querying SignedBeaconBlock and Beacon State", Label("unit"), func() {
	//	Context("When the slot is properly served by the beacon node", func() {
	//		It("Should provide a successful response.")
	//	})
	//	Context("When there is a skipped slot", func() {
	//		It("Should indicate that the slot was skipped")
	//		// Future use case.

	//	})
	//	Context("When the slot is not properly served", func() {
	//		It("Should return an error, and add the slot to the knownGaps table.")
	//	})
	//})

	//Describe("Receiving properly formatted Head SSE events.", Label("unit"), func() {
	//	Context("In sequential order", func() {
	//		It("Should write each event to the DB successfully.")
	//	})
	//	Context("With gaps in slots", func() {
	//		It("Should add the slots in between to the knownGaps table")
	//	})
	//	Context("With a repeat slot", func() {
	//		It("Should recognize the reorg and process it.")
	//	})
	//	Context("With the previousBlockHash not matching the parentBlockHash", func() {
	//		It("Should recognize the reorg and add the previous slot to knownGaps table.")
	//	})
	//	Context("Out of order", func() {
	//		It("Not sure what it should do....")
	//	})
	//	Context("With a skipped slot", func() {
	//		It("Should recognize the slot as skipped and continue without error.")
	//		// Future use case
	//	})
	//})
})

// Must run before each test. We can't use the beforeEach because of the way
// Ginko treats race conditions.
func setUpTest(testEvents map[string]*Message, protocol string, address string, port int, dummyParentRoot string,
	dbHost string, dbPort int, dbName string, dbUser string, dbPassword string, dbDriver string) *beaconclient.BeaconClient {

	BC := *beaconclient.CreateBeaconClient(context.Background(), protocol, address, port)
	DB, err := postgres.SetupPostgresDb(dbHost, dbPort, dbName, dbUser, dbPassword, dbDriver)
	Expect(err).ToNot(HaveOccurred())

	// Drop all records from the DB.
	clearEthclDbTables(DB)
	BC.Db = DB

	return &BC
}

// Wrapper function to send a head message to the beaconclient
func sendHeadMessage(bc *beaconclient.BeaconClient, head beaconclient.Head) {

	data, err := json.Marshal(head)
	Expect(err).ToNot(HaveOccurred())

	startInserts := atomic.LoadUint64(&bc.Metrics.HeadTrackingInserts)
	bc.HeadTracking.MessagesCh <- &sse.Event{
		ID:    []byte{},
		Data:  data,
		Event: []byte{},
		Retry: []byte{},
	}
	for atomic.LoadUint64(&bc.Metrics.HeadTrackingInserts) != startInserts+1 {
		time.Sleep(1 * time.Second)
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
	deleteQueries := []string{"DELETE FROM ethcl.slots;", "DELETE FROM ethcl.signed_beacon_block;", "DELETE FROM ethcl.beacon_state;", "DELETE FROM ethcl.batch_processing;"}
	for _, queries := range deleteQueries {
		_, err := db.Exec(context.Background(), queries)
		Expect(err).ToNot(HaveOccurred())
	}

}

type TestBeaconNode struct {
	TestEvents map[string]*Message
}

// Create a new new mock for the beacon node.
func SetupBeaconNodeMock(TestEvents map[string]*Message, protocol string, address string, port int, dummyParentRoot string) {
	httpmock.Activate()
	testBC := &TestBeaconNode{
		TestEvents: TestEvents,
	}

	stateUrl := `=~^` + protocol + "://" + address + ":" + strconv.Itoa(port) + beaconclient.BcStateQueryEndpoint + `([^/]+)\z`
	httpmock.RegisterResponder("GET", stateUrl,
		func(req *http.Request) (*http.Response, error) {
			// Get ID from request
			id := httpmock.MustGetSubmatch(req, 1)
			dat, err := testBC.provideSsz(id, "state", dummyParentRoot)
			if err != nil {
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
			dat, err := testBC.provideSsz(id, "block", dummyParentRoot)
			if err != nil {
				return httpmock.NewStringResponse(404, fmt.Sprintf("Unable to find file for %s", id)), err
			}
			return httpmock.NewBytesResponse(200, dat), nil
		},
	)
}

// A function to mimic querying the state from the beacon node. We simply get the SSZ file are return it.
func (tbc TestBeaconNode) provideSsz(slotIdentifier string, sszIdentifier string, dummyParentRoot string) ([]byte, error) {
	var slotFile string
	var Message *Message

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
			}
			return block.MarshalSSZ()
		}
		if sszIdentifier == "state" {
			dat, err := os.ReadFile(slotFile)
			if err != nil {
				return nil, fmt.Errorf("Can't find the slot file, %s", slotFile)
			}
			state := st.BeaconState{}
			state.UnmarshalSSZ(dat)
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
