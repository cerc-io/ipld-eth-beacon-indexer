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
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	si "github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	dt "github.com/prysmaticlabs/prysm/encoding/ssz/detect"
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
	ParentRoot  string // The parent root, leave it empty if you want a to use the universal
	ForkVersion string // Specify the fork version. This is needed as a workaround to create dummy SignedBeaconBlocks.
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
		dbPort                  int    = 8076
		dbName                  string = "vulcanize_testing"
		dbUser                  string = "vdbm"
		dbPassword              string = "password"
		dbDriver                string = "pgx"
		dummyParentRoot         string = "46f98c08b54a71dfda4d56e29ec3952b8300cd8d6b67a9b6c562ae96a7a25a42"
		knownGapsTableIncrement int    = 100000
		maxRetry                int    = 60
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
				TestNotes: "A block that is supposed to replicate slot 100, but contains some dummy test information.",
				MimicConfig: &MimicConfig{
					ForkVersion: "phase0",
				},
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
				TestNotes: "A block that is supposed to replicate slot 100, but contains some dummy test information.",
				MimicConfig: &MimicConfig{
					ForkVersion: "phase0",
				},
				SignedBeaconBlock: filepath.Join("ssz-data", "100", "signed-beacon-block.ssz"),
				BeaconState:       filepath.Join("ssz-data", "100", "beacon-state.ssz"),
			},
			"102-wrong-ssz-1": {
				HeadMessage: beaconclient.Head{
					Slot:                      "102",
					Block:                     "0x46f98c08b54a71dfda4d56e29ec3952b8300cd8d6b67a9b6c562ae96a7a25a42",
					State:                     "0x9b20b114c613c1aa462e02d590b3da902b0a1377e938ed0f94dd3491d763ef67",
					CurrentDutyDependentRoot:  "",
					PreviousDutyDependentRoot: "",
					EpochTransition:           false,
					ExecutionOptimistic:       false,
				},
				TestNotes:         "A bad block that returns the wrong ssz objects, used for testing incorrect SSZ decoding.",
				BeaconState:       filepath.Join("ssz-data", "102", "signed-beacon-block.ssz"),
				SignedBeaconBlock: filepath.Join("ssz-data", "102", "beacon-state.ssz"),
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
				TestNotes:         "An easy to process Phase 0 block",
				SignedBeaconBlock: filepath.Join("ssz-data", "101", "signed-beacon-block.ssz"),
				BeaconState:       filepath.Join("ssz-data", "101", "beacon-state.ssz"),
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
				TestNotes: "This is a dummy message that is used for reorgs",
				MimicConfig: &MimicConfig{
					ForkVersion: "altair",
				},
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
				TestNotes: "This is a dummy message that is used for reorgs",
				MimicConfig: &MimicConfig{
					ForkVersion: "altair",
				},
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
			"3797056": {
				HeadMessage: beaconclient.Head{
					Slot:                     "3797056",
					Block:                    "",
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

	Describe("Receiving New Head SSE messages", Label("unit", "behavioral"), func() {
		Context("Correctly formatted Phase0 Block", func() {
			It("Should turn it into a struct successfully.", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "99")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testProcessBlock(bc, BeaconNodeTester.TestEvents["100"].HeadMessage, 3, maxRetry, 1, 0, 0)
				validateSignedBeaconBlock(bc, BeaconNodeTester.TestEvents["100"].HeadMessage, "0x629ae1587895043076500f4f5dcb202a47c2fc95d5b5c548cb83bc97bd2dbfe1", "0x8d3f027beef5cbd4f8b29fc831aba67a5d74768edca529f5596f07fd207865e1", "/blocks/QHVAEQBQGQ4TKNJUGAYDGNZRGM2DOZJSGZTDMMLEG5QTIYTCMRQTKYRSGNTGCMDCGI2WINLGMM2DMNJRGYYGMMTBHEZGINJSME3DGYRZGE4WE")
				validateBeaconState(bc, BeaconNodeTester.TestEvents["100"].HeadMessage, "/blocks/QHVAEQRQPBTDEOBWMEYDGNZZMMYDGOBWMEZWGN3CMUZDQZBQGVSDQMRZMY4GKYRXMIZDQMDDMM4WKZDFGE2TINBZMFTDEMDFMJRWIMBWME3WCNJW")

			})
		})
		Context("Correctly formatted Altair Block", func() {
			It("Should turn it into a struct successfully.", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "2375702")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testProcessBlock(bc, BeaconNodeTester.TestEvents["2375703"].HeadMessage, 74240, maxRetry, 1, 0, 0)
				validateSignedBeaconBlock(bc, BeaconNodeTester.TestEvents["2375703"].HeadMessage, "0x83154c692b9cce50bdf56af5a933da0a020ed7ff809a6a8236301094c7f25276", "0xd74b1c60423651624de6bb301ac25808951c167ba6ecdd9b2e79b4315aee8202", "/blocks/QHVAEQRQPA2DGOJSGM3TEYZVMY3GKMZZGQ4TSZJTGFRGMOJSGQZTQODCGU4DCNJWGM4TCMBTGE2DSZRQMY2TIZRYME2DKMZXG4ZWEMJYGAZDGMBR")
				validateBeaconState(bc, BeaconNodeTester.TestEvents["2375703"].HeadMessage, "/blocks/QHVAEQRQPBRDMMRRGVRDKNRQGI3TGYLGGYZWKYZXMUYDCMJVG4ZGENRQMVRTCY3BGBRDAMRTGJTDQZTGGQ2GMY3EGRSWINJVMM3TKMRWMU4TMNDF")
			})
		})
		Context("Correctly formatted Altair Test Blocks", func() {
			It("Should turn it into a struct successfully.", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "2375702")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testProcessBlock(bc, BeaconNodeTester.TestEvents["2375703-dummy"].HeadMessage, 74240, maxRetry, 1, 0, 0)

				bc = setUpTest(BeaconNodeTester.TestConfig, "2375702")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testProcessBlock(bc, BeaconNodeTester.TestEvents["2375703-dummy-2"].HeadMessage, 74240, maxRetry, 1, 0, 0)

			})
		})
		Context("Correctly formatted Phase0 Test Blocks", func() {
			It("Should turn it into a struct successfully.", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "99")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testProcessBlock(bc, BeaconNodeTester.TestEvents["100-dummy"].HeadMessage, 3, maxRetry, 1, 0, 0)

				bc = setUpTest(BeaconNodeTester.TestConfig, "99")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testProcessBlock(bc, BeaconNodeTester.TestEvents["100-dummy-2"].HeadMessage, 3, maxRetry, 1, 0, 0)
			})

		})
		Context("Two consecutive correct blocks", func() {
			It("Should handle both blocks correctly, without any reorgs or known_gaps", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "99")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testProcessBlock(bc, BeaconNodeTester.TestEvents["100"].HeadMessage, 3, maxRetry, 1, 0, 0)
				BeaconNodeTester.testProcessBlock(bc, BeaconNodeTester.TestEvents["101"].HeadMessage, 3, maxRetry, 1, 0, 0)
			})
		})
		Context("Two consecutive blocks with a bad parent", func() {
			It("Should add the previous block to the knownGaps table.", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "99")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testProcessBlock(bc, BeaconNodeTester.TestEvents["100-dummy"].HeadMessage, 3, maxRetry, 1, 0, 0)
				BeaconNodeTester.testProcessBlock(bc, BeaconNodeTester.TestEvents["101"].HeadMessage, 3, maxRetry, 1, 1, 1)
			})
		})
		Context("Phase 0: We have a correctly formated SSZ SignedBeaconBlock and BeaconState", func() {
			It("Should be able to get each objects root hash.", func() {
				testSszRoot(BeaconNodeTester.TestEvents["100"])
			})
		})
		Context("Altair: We have a correctly formated SSZ SignedBeaconBlock and BeaconState", func() {
			It("Should be able to get each objects root hash.", func() {
				testSszRoot(BeaconNodeTester.TestEvents["2375703"])
			})
		})
		//Context("When there is a skipped slot", func() {
		//	It("Should indicate that the slot was skipped", func() {

		//	})
		//})
		Context("When the proper SSZ objects are not served", Label("now"), func() {
			It("Should return an error, and add the slot to the knownGaps table.", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "101")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testProcessBlock(bc, BeaconNodeTester.TestEvents["102-wrong-ssz-1"].HeadMessage, 3, maxRetry, 0, 1, 0)

				knownGapCount := countKnownGapsTable(bc.Db)
				Expect(knownGapCount).To(Equal(1))

				start, end := queryKnownGaps(bc.Db, "102", "102")
				Expect(start).To(Equal(102))
				Expect(end).To(Equal(102))
			})
		})
	})

	Describe("Known Gaps Scenario", Label("unit", "behavioral"), func() {
		Context("There is a gap at start up within one incrementing range.", func() {
			It("Should add only a single entry to the knownGaps table.", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "10")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testKnownGapsMessages(bc, 100, 1, maxRetry, BeaconNodeTester.TestEvents["100"].HeadMessage)
				start, end := queryKnownGaps(bc.Db, "11", "99")
				Expect(start).To(Equal(11))
				Expect(end).To(Equal(99))
			})
		})
		Context("There is a gap at start up spanning multiple incrementing range.", func() {
			It("Should add multiple entries to the knownGaps table.", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "5")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testKnownGapsMessages(bc, 10, 10, maxRetry, BeaconNodeTester.TestEvents["100"].HeadMessage)

				start, end := queryKnownGaps(bc.Db, "6", "16")
				Expect(start).To(Equal(6))
				Expect(end).To(Equal(16))

				start, end = queryKnownGaps(bc.Db, "96", "99")
				Expect(start).To(Equal(96))
				Expect(end).To(Equal(99))
			})
		})
		Context("Gaps between two head messages", func() {
			It("Should add the slots in-between", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "99")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testKnownGapsMessages(bc, 1000000, 3, maxRetry, BeaconNodeTester.TestEvents["100"].HeadMessage, BeaconNodeTester.TestEvents["2375703"].HeadMessage)

				start, end := queryKnownGaps(bc.Db, "101", "1000101")
				Expect(start).To(Equal(101))
				Expect(end).To(Equal(1000101))

				start, end = queryKnownGaps(bc.Db, "2000101", "2375702")
				Expect(start).To(Equal(2000101))
				Expect(end).To(Equal(2375702))
			})
		})
	})

	Describe("ReOrg Scenario", Label("unit", "behavioral"), func() {
		Context("Altair: Multiple head messages for the same slot.", func() {
			It("The previous block should be marked as 'forked', the new block should be the only one marked as 'proposed'.", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "2375702")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testMultipleHead(bc, TestEvents["2375703"].HeadMessage, TestEvents["2375703-dummy"].HeadMessage, 74240, maxRetry)
			})
		})
		Context("Phase0: Multiple head messages for the same slot.", func() {
			It("The previous block should be marked as 'forked', the new block should be the only one marked as 'proposed'.", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "99")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testMultipleHead(bc, TestEvents["100-dummy"].HeadMessage, TestEvents["100"].HeadMessage, 3, maxRetry)
			})
		})
		Context("Phase 0: Multiple reorgs have occurred on this slot", func() {
			It("The previous blocks should be marked as 'forked', the new block should be the only one marked as 'proposed'.", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "99")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testMultipleReorgs(bc, TestEvents["100-dummy"].HeadMessage, TestEvents["100-dummy-2"].HeadMessage, TestEvents["100"].HeadMessage, 3, maxRetry)
			})
		})
		Context("Altair: Multiple reorgs have occurred on this slot", Label("new"), func() {
			It("The previous blocks should be marked as 'forked', the new block should be the only one marked as 'proposed'.", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "2375702")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testMultipleReorgs(bc, TestEvents["2375703-dummy"].HeadMessage, TestEvents["2375703-dummy-2"].HeadMessage, TestEvents["2375703"].HeadMessage, 74240, maxRetry)
			})
		})
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
func setUpTest(config Config, maxSlot string) *beaconclient.BeaconClient {
	bc := *beaconclient.CreateBeaconClient(context.Background(), config.protocol, config.address, config.port)
	db, err := postgres.SetupPostgresDb(config.dbHost, config.dbPort, config.dbName, config.dbUser, config.dbPassword, config.dbDriver)
	Expect(err).ToNot(HaveOccurred())

	// Drop all records from the DB.
	clearEthclDbTables(db)

	// Add an slot to the ethcl.slots table so it we can control how known_gaps are handled.
	writeSlot(db, maxSlot)
	bc.Db = db

	return &bc
}

// A helper function to validate the expected output from the ethcl.slots table.
func validateSlot(bc *beaconclient.BeaconClient, headMessage beaconclient.Head, correctEpoch int, correctStatus string) {
	epoch, dbSlot, blockRoot, stateRoot, status := queryDbSlotAndBlock(bc.Db, headMessage.Slot, headMessage.Block)
	baseSlot, err := strconv.Atoi(headMessage.Slot)
	Expect(err).ToNot(HaveOccurred())
	Expect(dbSlot).To(Equal(baseSlot))
	Expect(epoch).To(Equal(correctEpoch))
	Expect(blockRoot).To(Equal(headMessage.Block))
	Expect(stateRoot).To(Equal(headMessage.State))
	Expect(status).To(Equal(correctStatus))
}

// A helper function to validate the expected output from the ethcl.signed_beacon_block table.
func validateSignedBeaconBlock(bc *beaconclient.BeaconClient, headMessage beaconclient.Head, correctParentRoot string, correctEth1BlockHash string, correctMhKey string) {
	dbSlot, blockRoot, parentRoot, eth1BlockHash, mhKey := queryDbSignedBeaconBlock(bc.Db, headMessage.Slot, headMessage.Block)
	baseSlot, err := strconv.Atoi(headMessage.Slot)
	Expect(err).ToNot(HaveOccurred())
	Expect(dbSlot).To(Equal(baseSlot))
	Expect(blockRoot).To(Equal(headMessage.Block))
	Expect(parentRoot, correctParentRoot)
	Expect(eth1BlockHash, correctEth1BlockHash)
	Expect(mhKey, correctMhKey)

}

// A helper function to validate the expected output from the ethcl.beacon_state table.
func validateBeaconState(bc *beaconclient.BeaconClient, headMessage beaconclient.Head, correctMhKey string) {
	dbSlot, stateRoot, mhKey := queryDbBeaconState(bc.Db, headMessage.Slot, headMessage.State)
	baseSlot, err := strconv.Atoi(headMessage.Slot)
	Expect(err).ToNot(HaveOccurred())
	Expect(dbSlot).To(Equal(baseSlot))
	Expect(stateRoot).To(Equal(headMessage.State))
	Expect(mhKey, correctMhKey)

}

// Wrapper function to send a head message to the beaconclient
func sendHeadMessage(bc *beaconclient.BeaconClient, head beaconclient.Head, maxRetry int, expectedSuccessfulInserts uint64) {

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
	for atomic.LoadUint64(&bc.Metrics.HeadTrackingInserts) != startInserts+expectedSuccessfulInserts {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			log.WithFields(log.Fields{
				"startInsert":  startInserts,
				"currentValue": atomic.LoadUint64(&bc.Metrics.HeadTrackingInserts),
			}).Error("HeadTracking Insert wasn't incremented properly.")
			Fail("Too many retries have occurred.")
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

// A helper function to query the ethcl.signed_beacon_block table based on the slot and block_root.
func queryDbSignedBeaconBlock(db sql.Database, querySlot string, queryBlockRoot string) (int, string, string, string, string) {
	sqlStatement := `SELECT slot, block_root, parent_block_root, eth1_block_hash, mh_key FROM ethcl.signed_beacon_block WHERE slot=$1 AND block_root=$2;`
	var slot int
	var blockRoot, parent_block_root, eth1_block_hash, mh_key string
	row := db.QueryRow(context.Background(), sqlStatement, querySlot, queryBlockRoot)
	err := row.Scan(&slot, &blockRoot, &parent_block_root, &eth1_block_hash, &mh_key)
	Expect(err).ToNot(HaveOccurred())
	return slot, blockRoot, parent_block_root, eth1_block_hash, mh_key
}

// A helper function to query the ethcl.signed_beacon_block table based on the slot and block_root.
func queryDbBeaconState(db sql.Database, querySlot string, queryStateRoot string) (int, string, string) {
	sqlStatement := `SELECT slot, state_root, mh_key FROM ethcl.beacon_state WHERE slot=$1 AND state_root=$2;`
	var slot int
	var stateRoot, mh_key string
	row := db.QueryRow(context.Background(), sqlStatement, querySlot, queryStateRoot)
	err := row.Scan(&slot, &stateRoot, &mh_key)
	Expect(err).ToNot(HaveOccurred())
	return slot, stateRoot, mh_key
}

// Count the entries in the knownGaps table.
func countKnownGapsTable(db sql.Database) int {
	var count int
	sqlStatement := "SELECT COUNT(*) FROM ethcl.known_gaps"
	err := db.QueryRow(context.Background(), sqlStatement).Scan(&count)
	Expect(err).ToNot(HaveOccurred())
	return count
}

// Return the start and end slot
func queryKnownGaps(db sql.Database, queryStartGap string, QueryEndGap string) (int, int) {
	sqlStatement := `SELECT start_slot, end_slot FROM ethcl.known_gaps WHERE start_slot=$1 AND end_slot=$2;`
	var startGap, endGap int
	row := db.QueryRow(context.Background(), sqlStatement, queryStartGap, QueryEndGap)
	err := row.Scan(&startGap, &endGap)
	Expect(err).ToNot(HaveOccurred())
	return startGap, endGap

}

// A function that will remove all entries from the ethcl tables for you.
func clearEthclDbTables(db sql.Database) {
	deleteQueries := []string{"DELETE FROM ethcl.slots;", "DELETE FROM ethcl.signed_beacon_block;", "DELETE FROM ethcl.beacon_state;", "DELETE FROM ethcl.known_gaps;"}
	for _, queries := range deleteQueries {
		_, err := db.Exec(context.Background(), queries)
		Expect(err).ToNot(HaveOccurred())
	}
}

// Write an entry to the ethcl.slots table with just a slot number
func writeSlot(db sql.Database, slot string) {
	_, err := db.Exec(context.Background(), beaconclient.UpsertSlotsStmt, "0", slot, "", "", "")
	Expect(err).ToNot(HaveOccurred())
}

// Read a file with the SignedBeaconBlock in SSZ and return the SSZ object. This is used for testing only.
// We can't use the readSignedBeaconBlockInterface to update struct fields so this is the workaround.
func readSignedBeaconBlock(slotFile string) (*st.SignedBeaconBlock, error) {
	dat, err := os.ReadFile(slotFile)
	if err != nil {
		return nil, fmt.Errorf("Can't find the slot file, %s", slotFile)
	}
	block := &st.SignedBeaconBlock{}
	err = block.UnmarshalSSZ(dat)
	Expect(err).ToNot(HaveOccurred())
	return block, nil
}

// Read a file with the SignedBeaconBlock in SSZ and return the SSZ object. This is used for testing only.
// We can't use the readSignedBeaconBlockInterface to update struct fields so this is the workaround.
func readSignedBeaconBlockAltair(slotFile string) (*st.SignedBeaconBlockAltair, error) {
	dat, err := os.ReadFile(slotFile)
	if err != nil {
		return nil, fmt.Errorf("Can't find the slot file, %s", slotFile)
	}
	block := &st.SignedBeaconBlockAltair{}
	err = block.UnmarshalSSZ(dat)
	Expect(err).ToNot(HaveOccurred())
	return block, nil
}

// Read a file with the SignedBeaconBlock in SSZ and return the SSZ objects interface. This is production like.
// It will provide the correct struct for the given fork.
func readSignedBeaconBlockInterface(slotFile string, vm *dt.VersionedUnmarshaler) (si.SignedBeaconBlock, error) {
	dat, err := os.ReadFile(slotFile)
	if err != nil {
		return nil, fmt.Errorf("Can't find the slot file, %s", slotFile)
	}

	block, err := vm.UnmarshalBeaconBlock(dat)
	Expect(err).ToNot(HaveOccurred())
	return block, nil

}

// Read a file with the BeaconState in SSZ and return the SSZ object
func readBeaconState(slotFile string) (state.BeaconState, *dt.VersionedUnmarshaler, error) {
	dat, err := os.ReadFile(slotFile)
	if err != nil {
		return nil, nil, fmt.Errorf("Can't find the slot file, %s", slotFile)
	}
	versionedUnmarshaler, err := dt.FromState(dat)
	Expect(err).ToNot(HaveOccurred())
	state, err := versionedUnmarshaler.UnmarshalBeaconState(dat)
	Expect(err).ToNot(HaveOccurred())
	return state, versionedUnmarshaler, nil
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
			// A dirty solution to handle different Block Types.
			// * I was unsuccessful in implementing generics.
			// * I can't use the interfaces.SignedBeaconBlock
			// * I was short on time.
			// * This solution allows us to hardcode the version and create the write block type for it when we
			// Are mimicing an existing block.
			switch Message.MimicConfig.ForkVersion {
			case "phase0":
				block, err := readSignedBeaconBlock(slotFile)
				if err != nil {
					return nil, err
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
			case "altair":
				block, err := readSignedBeaconBlockAltair(slotFile)
				if err != nil {
					return nil, err
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
		}
		if sszIdentifier == "state" {
			state, _, err := readBeaconState(slotFile)
			if err != nil {
				return nil, err
			}
			slot, err := strconv.ParseUint(Message.HeadMessage.Slot, 10, 64)
			Expect(err).ToNot(HaveOccurred())
			err = state.SetSlot(types.Slot(slot))
			Expect(err).ToNot(HaveOccurred())
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
func (tbc TestBeaconNode) testMultipleReorgs(bc *beaconclient.BeaconClient, firstHead beaconclient.Head, secondHead beaconclient.Head, thirdHead beaconclient.Head, epoch int, maxRetry int) {
	go bc.CaptureHead(tbc.TestConfig.knownGapsTableIncrement)
	time.Sleep(1 * time.Second)

	log.Info("Sending Phase0 Messages to BeaconClient")
	sendHeadMessage(bc, firstHead, maxRetry, 1)
	sendHeadMessage(bc, secondHead, maxRetry, 1)
	sendHeadMessage(bc, thirdHead, maxRetry, 1)

	curRetry := 0
	for atomic.LoadUint64(&bc.Metrics.HeadTrackingReorgs) != 2 {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			Fail(" Too many retries have occurred.")
		}
	}

	log.Info("Checking to make sure the fork was marked properly.")
	validateSlot(bc, firstHead, epoch, "forked")
	validateSlot(bc, secondHead, epoch, "forked")
	validateSlot(bc, thirdHead, epoch, "proposed")

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
			Fail("Too many retries have occurred.")
		}
	}

	if bc.Metrics.HeadTrackingKnownGaps != 0 {
		Fail("We found gaps when processing a single block")
	}

	log.Info("Make sure the forks were properly updated!")

	validateSlot(bc, firstHead, epoch, "forked")
	validateSlot(bc, secondHead, epoch, "proposed")
	validateSlot(bc, thirdHead, epoch, "forked")

}

// A test to validate a single block was processed correctly
func (tbc TestBeaconNode) testProcessBlock(bc *beaconclient.BeaconClient, head beaconclient.Head, epoch int, maxRetry int, expectedSuccessInsert uint64, expectedKnownGaps uint64, expectedReorgs uint64) {
	go bc.CaptureHead(tbc.TestConfig.knownGapsTableIncrement)
	time.Sleep(1 * time.Second)
	sendHeadMessage(bc, head, maxRetry, expectedSuccessInsert)

	curRetry := 0
	for atomic.LoadUint64(&bc.Metrics.HeadTrackingKnownGaps) != expectedKnownGaps {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			Fail(fmt.Sprintf("Wrong gap metrics, got: %d, wanted %d", bc.Metrics.HeadTrackingKnownGaps, expectedKnownGaps))
		}
	}

	curRetry = 0
	for atomic.LoadUint64(&bc.Metrics.HeadTrackingReorgs) != expectedReorgs {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			Fail(fmt.Sprintf("Wrong reorg metrics, got: %d, wanted %d", bc.Metrics.HeadTrackingKnownGaps, expectedKnownGaps))
		}
	}

	if expectedSuccessInsert > 0 {
		validateSlot(bc, head, epoch, "proposed")
	}
}

// A test that ensures that if two HeadMessages occur for a single slot they are marked
// as proposed and forked correctly.
func (tbc TestBeaconNode) testMultipleHead(bc *beaconclient.BeaconClient, firstHead beaconclient.Head, secondHead beaconclient.Head, epoch int, maxRetry int) {
	go bc.CaptureHead(tbc.TestConfig.knownGapsTableIncrement)
	time.Sleep(1 * time.Second)

	sendHeadMessage(bc, firstHead, maxRetry, 1)
	sendHeadMessage(bc, secondHead, maxRetry, 1)

	curRetry := 0
	for atomic.LoadUint64(&bc.Metrics.HeadTrackingReorgs) != 1 {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			Fail(" Too many retries have occurred.")
		}
	}

	if bc.Metrics.HeadTrackingKnownGaps != 0 {
		Fail("We found gaps when processing a single block")
	}

	log.Info("Checking Altair to make sure the fork was marked properly.")
	validateSlot(bc, firstHead, epoch, "forked")
	validateSlot(bc, secondHead, epoch, "proposed")
}

// A test that ensures that if two HeadMessages occur for a single slot they are marked
// as proposed and forked correctly.
func (tbc TestBeaconNode) testKnownGapsMessages(bc *beaconclient.BeaconClient, tableIncrement int, expectedEntries uint64, maxRetry int, msg ...beaconclient.Head) {
	go bc.CaptureHead(tableIncrement)
	time.Sleep(1 * time.Second)

	for _, headMsg := range msg {
		sendHeadMessage(bc, headMsg, maxRetry, 1)
	}

	curRetry := 0
	for atomic.LoadUint64(&bc.Metrics.HeadTrackingKnownGaps) != expectedEntries {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			Fail("Too many retries have occurred.")
		}
	}

	log.Info("Checking to make sure we have the expected number of entries in the knownGaps table.")
	knownGapCount := countKnownGapsTable(bc.Db)
	Expect(knownGapCount).To(Equal(int(expectedEntries)))

	if atomic.LoadUint64(&bc.Metrics.HeadTrackingReorgs) != 0 {
		Fail("We found reorgs when we didn't expect it")
	}
}

// This function will make sure we are properly able to get the SszRoot of the SignedBeaconBlock and the BeaconState.
func testSszRoot(msg Message) {
	state, vm, err := readBeaconState(msg.BeaconState)
	Expect(err).ToNot(HaveOccurred())
	stateRoot, err := state.HashTreeRoot(context.Background())
	Expect(err).ToNot(HaveOccurred())
	Expect(msg.HeadMessage.State).To(Equal("0x" + hex.EncodeToString(stateRoot[:])))

	block, err := readSignedBeaconBlockInterface(msg.SignedBeaconBlock, vm)
	Expect(err).ToNot(HaveOccurred())
	blockRoot, err := block.Block().HashTreeRoot()
	Expect(err).ToNot(HaveOccurred())
	Expect(msg.HeadMessage.Block).To(Equal("0x" + hex.EncodeToString(blockRoot[:])))
}
