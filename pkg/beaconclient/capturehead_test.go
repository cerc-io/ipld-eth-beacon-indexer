package beaconclient_test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/sirupsen/logrus"

	//	. "github.com/onsi/gomega"
	"github.com/r3labs/sse/v2"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/beaconclient"
)

type Message struct {
	HeadMessage       beaconclient.Head       // The head messsage that will be streamed to the BeaconClient
	ReorgMessage      beaconclient.ChainReorg // The reorg messsage that will be streamed to the BeaconClient
	TestNotes         string                  // A small explanation of the purpose this structure plays in the testing landscape.
	SignedBeaconBlock string                  // The file path output of an SSZ encoded SignedBeaconBlock.
	BeaconState       string                  // The file path output of an SSZ encoded BeaconState.
	SuccessfulDBQuery string                  // A string that indicates what a query to the DB should output to pass the test.
}

var TestEvents map[string]*Message

var _ = Describe("Capturehead", func() {
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

	// We might also want to add an integration test that will actually process a single event, then end.
	// This will help us know that our models match that actual data being served from the beacon node.

	Describe("Receiving New Head SSE messages", Label("unit"), func() {
		Context("Correctly formatted", Label("dry"), func() {
			It("Should turn it into a struct successfully.", func() {
				server := createSseServer()
				logrus.Info("DONE!")
				client := sse.NewClient("http://localhost:8080" + beaconclient.BcHeadTopicEndpoint)

				logrus.Info("DONE!")
				ch := make(chan *sse.Event)
				go client.SubscribeChanRaw(ch)

				time.Sleep(2 * time.Second)
				logrus.Info("DONE!")
				sendMessageToStream(server, []byte("hello"))
				client.Unsubscribe(ch)
				val := <-ch

				logrus.Info("DONE!")
				logrus.Info(val)
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

// Create a new Sse.Server.
func createSseServer() *sse.Server {
	// server := sse.New()
	// server.CreateStream("")

	mux := http.NewServeMux()
	//mux.HandleFunc(beaconclient.BcHeadTopicEndpoint, func(w http.ResponseWriter, r *http.Request) {
	//	go func() {
	//		// Received Browser Disconnection
	//		<-r.Context().Done()
	//		println("The client is disconnected here")
	//		return
	//	}()

	//	server.ServeHTTP(w, r)
	//})
	mux.HandleFunc(beaconclient.BcStateQueryEndpoint, provideState)
	mux.HandleFunc(beaconclient.BcBlockQueryEndpoint, provideBlock)
	go http.ListenAndServe(":8080", mux)
	return server
}

// Send messages to the stream.
func sendMessageToStream(server *sse.Server, data []byte) {
	server.Publish("", &sse.Event{
		Data: data,
	})
	logrus.Info("publish complete")
}

// A function to mimic querying the state from the beacon node. We simply get the SSZ file are return it.
func provideState(w http.ResponseWriter, req *http.Request) {
	path := strings.Split(req.URL.Path, "/")
	slot := path[len(path)-1]
	slotFile := "data/" + slot + "/beacon-state.ssz"
	dat, err := os.ReadFile(slotFile)
	if err != nil {
		fmt.Fprintf(w, "Can't find the slot file, %s", slotFile)
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(dat)
}

// A function to mimic querying the state from the beacon node. We simply get the SSZ file are return it.
func provideBlock(w http.ResponseWriter, req *http.Request) {
	path := strings.Split(req.URL.Path, "/")
	slot := path[len(path)-1]
	slotFile := "data/" + slot + "/signed-beacon-block.ssz"
	dat, err := os.ReadFile(slotFile)
	if err != nil {
		fmt.Fprintf(w, "Can't find the slot file, %s", slotFile)
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(dat)
}
