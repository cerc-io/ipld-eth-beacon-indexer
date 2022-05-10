package beaconclient_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo/v2"
	"github.com/r3labs/sse"

	. "github.com/onsi/gomega"

	"github.com/vulcanize/ipld-ethcl-indexer/pkg/beaconclient"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql/postgres"
)

type Message struct {
	HeadMessage       beaconclient.Head       // The head messsage that will be streamed to the BeaconClient
	ReorgMessage      beaconclient.ChainReorg // The reorg messsage that will be streamed to the BeaconClient
	TestNotes         string                  // A small explanation of the purpose this structure plays in the testing landscape.
	SignedBeaconBlock string                  // The file path output of an SSZ encoded SignedBeaconBlock.
	BeaconState       string                  // The file path output of an SSZ encoded BeaconState.
}

var _ = Describe("Capturehead", func() {

	var (
		BC         beaconclient.BeaconClient
		DB         sql.Database
		err        error
		address    string = "localhost"
		port       int    = 8080
		protocol   string = "http"
		TestEvents map[string]*Message
		dbHost     string = "localhost"
		dbPort     int    = 8077
		dbName     string = "vulcanize_testing"
		dbUser     string = "vdbm"
		dbPassword string = "password"
		dbDriver   string = "pgx"
	)

	BeforeEach(func() {
		TestEvents = map[string]*Message{
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
				TestNotes:         "This is a simple, easy to process block.",
				SignedBeaconBlock: filepath.Join("data", "100", "signed-beacon-block.ssz"),
				BeaconState:       filepath.Join("data", "100", "beacon-state.ssz"),
			},
			"101": {
				HeadMessage: beaconclient.Head{
					Slot:                      "101",
					Block:                     "0xabe1a972e512182d04f0d4a5c9c25f9ee57c2e9d0ff3f4c4c82fd42d13d31083",
					State:                     "0xcb04aa2edbf13c7bb7e7bd9b621ced6832e0075e89147352eac3019a824ce847",
					CurrentDutyDependentRoot:  "",
					PreviousDutyDependentRoot: "",
					EpochTransition:           false,
					ExecutionOptimistic:       false,
				},
				TestNotes:         "This is a simple, easy to process block.",
				SignedBeaconBlock: filepath.Join("data", "101", "signed-beacon-block.ssz"),
				BeaconState:       filepath.Join("data", "101", "beacon-state.ssz"),
			},
		}
		SetupBeaconNodeMock(TestEvents, protocol, address, port)
		BC = *beaconclient.CreateBeaconClient(context.Background(), protocol, address, port)
		DB, err = postgres.SetupPostgresDb(dbHost, dbPort, dbName, dbUser, dbPassword, dbDriver)
		Expect(err).ToNot(HaveOccurred())
		clearEthclDbTables(DB)
		// Drop all records from the DB.
		BC.Db = DB

	})

	// We might also want to add an integration test that will actually process a single event, then end.
	// This will help us know that our models match that actual data being served from the beacon node.

	Describe("Receiving New Head SSE messages", Label("unit"), func() {
		Context("Correctly formatted", func() {
			It("Should turn it into a struct successfully.", func() {
				go BC.CaptureHead()
				sendHeadMessage(BC, TestEvents["100"].HeadMessage)

				epoch, slot, blockRoot, stateRoot, status := queryDbSlotAndBlock(DB, TestEvents["100"].HeadMessage.Slot, TestEvents["100"].HeadMessage.Block)
				Expect(slot).To(Equal(100))
				Expect(epoch).To(Equal(3))
				Expect(blockRoot).To(Equal(TestEvents["100"].HeadMessage.Block))
				Expect(stateRoot).To(Equal(TestEvents["100"].HeadMessage.State))
				Expect(status).To(Equal("proposed"))

			})
		})
		//Context("A single incorrectly formatted", func() {
		//	It("Should create an error, maybe also add the projected slot to the knownGaps table......")
		//})
		//Context("An incorrectly formatted message sandwiched between correctly formatted messages", func() {
		//	It("Should create an error, maybe also add the projected slot to the knownGaps table......")
		//})
	})

	//Describe("Receiving New Reorg SSE messages", Label("unit"), func() {
	//	Context("Reorg slot is already in the DB", func() {
	//		It("The previous block should be marked as 'forked', the new block should be the only one marked as 'proposed'.")
	//	})
	//	Context("Multiple reorgs have occurred on this slot", func() {
	//		It("The previous blocks should be marked as 'forked', the new block should be the only one marked as 'proposed'.")
	//	})
	//	Context("Reorg slot in not already in the DB", func() {
	//		It("Should simply have the correct slot in the DB.")
	//	})

	//})

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

// Wrapper function to send a head message to the beaconclient
func sendHeadMessage(bc beaconclient.BeaconClient, head beaconclient.Head) {

	data, err := json.Marshal(head)
	Expect(err).ToNot(HaveOccurred())

	bc.HeadTracking.MessagesCh <- &sse.Event{
		ID:    []byte{},
		Data:  data,
		Event: []byte{},
		Retry: []byte{},
	}
	time.Sleep(2 * time.Second)
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
func SetupBeaconNodeMock(TestEvents map[string]*Message, protocol string, address string, port int) {
	httpmock.Activate()
	testBC := &TestBeaconNode{
		TestEvents: TestEvents,
	}

	stateUrl := `=~^` + protocol + "://" + address + ":" + strconv.Itoa(port) + beaconclient.BcStateQueryEndpoint + `([^/]+)\z`
	httpmock.RegisterResponder("GET", stateUrl,
		func(req *http.Request) (*http.Response, error) {
			// Get ID from request
			id := httpmock.MustGetSubmatch(req, 1)
			dat, err := testBC.provideSsz(id, "state")
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
			dat, err := testBC.provideSsz(id, "block")
			if err != nil {
				return httpmock.NewStringResponse(404, fmt.Sprintf("Unable to find file for %s", id)), err
			}
			return httpmock.NewBytesResponse(200, dat), nil
		},
	)
}

// A function to mimic querying the state from the beacon node. We simply get the SSZ file are return it.
func (tbc TestBeaconNode) provideSsz(slotIdentifier string, sszIdentifier string) ([]byte, error) {
	var slotFile string
	for _, val := range tbc.TestEvents {
		if sszIdentifier == "state" {
			if val.HeadMessage.Slot == slotIdentifier || val.HeadMessage.State == slotIdentifier {
				slotFile = val.BeaconState
			}
		} else if sszIdentifier == "block" {
			if val.HeadMessage.Slot == slotIdentifier || val.HeadMessage.Block == slotIdentifier {
				slotFile = val.SignedBeaconBlock
			}
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
