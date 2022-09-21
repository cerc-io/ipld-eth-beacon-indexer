// VulcanizeDB
// Copyright © 2022 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
package beaconclient_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo/v2"
	"github.com/r3labs/sse"
	log "github.com/sirupsen/logrus"

	. "github.com/onsi/gomega"

	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/beaconclient"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/database/sql/postgres"
)

var (
	address                 string = "localhost"
	port                    int    = 8080
	protocol                string = "http"
	dbHost                  string = "localhost"
	dbPort                  int    = 8076
	dbName                  string = "vulcanize_testing"
	dbUser                  string = "vdbm"
	dbPassword              string = "password"
	dbDriver                string = "pgx"
	bcUniqueIdentifier      int    = 100
	dummyParentRoot         string = "46f98c08b54a71dfda4d56e29ec3952b8300cd8d6b67a9b6c562ae96a7a25a42"
	knownGapsTableIncrement int    = 100000
	maxRetry                int    = 160

	TestEvents = map[string]Message{
		"0": {
			HeadMessage: beaconclient.Head{
				Slot:                      "0",
				Block:                     "0x4d611d5b93fdab69013a7f0a2f961caca0c853f87cfe9595fe50038163079360",
				State:                     "0x7e76880eb67bbdc86250aa578958e9d0675e64e714337855204fb5abaaf82c2b",
				CurrentDutyDependentRoot:  "",
				PreviousDutyDependentRoot: "",
				EpochTransition:           false,
				ExecutionOptimistic:       false,
			},
			SignedBeaconBlock:             filepath.Join("ssz-data", "0", "signed-beacon-block.ssz"),
			BeaconState:                   filepath.Join("ssz-data", "0", "beacon-state.ssz"),
			CorrectSignedBeaconBlockMhKey: "/blocks/QLVAEQRQPA2GINRRGFSDKYRZGNTGIYLCGY4TAMJTME3WMMDBGJTDSNRRMNQWGYJQMM4DKM3GHA3WGZTFHE2TSNLGMU2TAMBTHAYTMMZQG44TGNRQ",
			CorrectBeaconStateMhKey:       "/blocks/QLVAEQRQPA3WKNZWHA4DAZLCGY3WEYTEMM4DMMRVGBQWCNJXHA4TKODFHFSDANRXGVSTMNDFG4YTIMZTG44DKNJSGA2GMYRVMFRGCYLGHAZGGMTC",
			CorrectParentRoot:             "0x0000000000000000000000000000000000000000000000000000000000000000",
			CorrectEth1DataBlockHash:      "0x0000000000000000000000000000000000000000000000000000000000000000",
		},
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
			TestNotes:                     "An easy to process Phase 0 block",
			SignedBeaconBlock:             filepath.Join("ssz-data", "100", "signed-beacon-block.ssz"),
			BeaconState:                   filepath.Join("ssz-data", "100", "beacon-state.ssz"),
			CorrectSignedBeaconBlockMhKey: "/blocks/QLVAEQRQPA2TQMRRHA3WKOJXMY3TKMRQMJRDMOLFMVQTAMJUMMZTQMZUMM4TMNDDGQ2TENJZGM3TEYJQMVQWCZLBGNTDAMZSGAYTGNZZG44TSNTC",
			CorrectBeaconStateMhKey:       "/blocks/QLVAEQRQPBTDEOBWMEYDGNZZMMYDGOBWMEZWGN3CMUZDQZBQGVSDQMRZMY4GKYRXMIZDQMDDMM4WKZDFGE2TINBZMFTDEMDFMJRWIMBWME3WCNJW",
			CorrectParentRoot:             "0x629ae1587895043076500f4f5dcb202a47c2fc95d5b5c548cb83bc97bd2dbfe1",
			CorrectEth1DataBlockHash:      "0x8d3f027beef5cbd4f8b29fc831aba67a5d74768edca529f5596f07fd207865e1",
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
			TestNotes:                     "An easy to process Phase 0 block",
			SignedBeaconBlock:             filepath.Join("ssz-data", "101", "signed-beacon-block.ssz"),
			BeaconState:                   filepath.Join("ssz-data", "101", "beacon-state.ssz"),
			CorrectEth1DataBlockHash:      "0x8d3f027beef5cbd4f8b29fc831aba67a5d74768edca529f5596f07fd207865e1",
			CorrectSignedBeaconBlockMhKey: "/blocks/QLVAEQRQPBQWEZJRME4TOMTFGUYTEMJYGJSDANDGGBSDIYJVMM4WGMRVMY4WKZJVG5RTEZJZMQYGMZRTMY2GGNDDHAZGMZBUGJSDCM3EGMYTAOBT",
			CorrectBeaconStateMhKey:       "/blocks/QLVAEQRQPBRWEMBUMFQTEZLEMJTDCM3DG5RGEN3FG5RGIOLCGYZDCY3FMQ3DQMZSMUYDANZVMU4DSMJUG4ZTKMTFMFRTGMBRHFQTQMRUMNSTQNBX",
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
				Slot:                      "2375703",
				Block:                     "0x4392372c5f6e39499e31bf924388b5815639103149f0f54f8a453773b1802301",
				State:                     "0xb6215b560273af63ec7e011572b60ec1ca0b0232f8ff44fcd4ed55c7526e964e",
				CurrentDutyDependentRoot:  "",
				PreviousDutyDependentRoot: "",
				EpochTransition:           false,
				ExecutionOptimistic:       false,
			},
			TestNotes:                     "An easy to process Altair Block",
			SignedBeaconBlock:             filepath.Join("ssz-data", "2375703", "signed-beacon-block.ssz"),
			BeaconState:                   filepath.Join("ssz-data", "2375703", "beacon-state.ssz"),
			CorrectEth1DataBlockHash:      "0xd74b1c60423651624de6bb301ac25808951c167ba6ecdd9b2e79b4315aee8202",
			CorrectParentRoot:             "0x08736ddc20b77f65d1aa6301f7e6e856a820ff3ce6430ed2c3694ae35580e740",
			CorrectSignedBeaconBlockMhKey: "/blocks/QLVAEQRQPA2DGOJSGM3TEYZVMY3GKMZZGQ4TSZJTGFRGMOJSGQZTQODCGU4DCNJWGM4TCMBTGE2DSZRQMY2TIZRYME2DKMZXG4ZWEMJYGAZDGMBR",
			CorrectBeaconStateMhKey:       "/blocks/QLVAEQRQPBRDMMRRGVRDKNRQGI3TGYLGGYZWKYZXMUYDCMJVG4ZGENRQMVRTCY3BGBRDAMRTGJTDQZTGGQ2GMY3EGRSWINJVMM3TKMRWMU4TMNDF",
		},
		"3797056": {
			HeadMessage: beaconclient.Head{
				Slot:                      "3797056",
				Block:                     "",
				State:                     "",
				CurrentDutyDependentRoot:  "",
				PreviousDutyDependentRoot: "",
				EpochTransition:           false,
				ExecutionOptimistic:       false,
			},
			TestNotes: "An easy to process Altair Block",
			// The file below should not exist, this will trigger an error message and 404 response from the mock.
			SignedBeaconBlock: filepath.Join("ssz-data", "3797056", "should-not-exist.txt"),
			BeaconState:       filepath.Join("ssz-data", "3797056", "beacon-state.ssz"),
		},
		"4636671": {
			HeadMessage: beaconclient.Head{
				Slot:                      "4636671",
				Block:                     "0xe7d4f3b7924c30ae047fceabb853b8afdae32b85e0a87ab6c4c37421b353a1da",
				State:                     "0x66146a0bc8656a63aaf5dd357f327cac58c83fc90582ced82bebcc6e5f11855b",
				CurrentDutyDependentRoot:  "",
				PreviousDutyDependentRoot: "",
				EpochTransition:           false,
				ExecutionOptimistic:       false,
			},
			TestNotes:                     "The last Altair block",
			SignedBeaconBlock:             filepath.Join("ssz-data", "4636671", "signed-beacon-block.ssz"),
			BeaconState:                   filepath.Join("ssz-data", "4636671", "beacon-state.ssz"),
			CorrectEth1DataBlockHash:      "0xa5b11e0cfb9ffd53e298f0d24fe07bc7a19ada6e52fa3f09397e1b34c07b4ec6",
			CorrectParentRoot:             "0x47fc3b7a28512a2570438c02bd0b96ebcac8bbcd97eed6d50f15454f37ac51b8",
			CorrectSignedBeaconBlockMhKey: "",
			CorrectBeaconStateMhKey:       "",
		},
		"4636672": {
			HeadMessage: beaconclient.Head{
				Slot:                      "4636672",
				Block:                     "0x9429ce339da8944dd2e1565be8cac5bf634cae2120b6937c081e39148a7f4b1a",
				State:                     "0x0067a5d28b38e6e2f59a73046fabbf16a782b978c2c89621a679e7f682b05bd4",
				CurrentDutyDependentRoot:  "",
				PreviousDutyDependentRoot: "",
				EpochTransition:           true,
				ExecutionOptimistic:       false,
			},
			TestNotes:                     "The first Bellatrix block (empty ExecutionPayload)",
			SignedBeaconBlock:             filepath.Join("ssz-data", "4636672", "signed-beacon-block.ssz"),
			BeaconState:                   filepath.Join("ssz-data", "4636672", "beacon-state.ssz"),
			CorrectEth1DataBlockHash:      "0x3b7d392e46db19704d677cadb3310c3776d8c0b8cb2af1c324bb4a394b7f8164",
			CorrectParentRoot:             "0xe7d4f3b7924c30ae047fceabb853b8afdae32b85e0a87ab6c4c37421b353a1da",
			CorrectSignedBeaconBlockMhKey: "/blocks/QLVAEQRQPA4TIMRZMNSTGMZZMRQTQOJUGRSGIMTFGE2TMNLCMU4GGYLDGVRGMNRTGRRWCZJSGEZDAYRWHEZTOYZQHAYWKMZZGE2DQYJXMY2GEMLB",
			CorrectBeaconStateMhKey:       "",
		},
		"4700013": {
			HeadMessage: beaconclient.Head{
				Slot:                      "4700013",
				Block:                     "0x810a00400a80cdffc11ffdcf17ac404ac4dba215b95221955a9dfddf163d0b0d",
				State:                     "0x171ef131e0638eddfe1ef73e7b483e344b1cf128b092f2c39e946eb7775b3a2f",
				CurrentDutyDependentRoot:  "",
				PreviousDutyDependentRoot: "",
				EpochTransition:           true,
				ExecutionOptimistic:       false,
			},
			TestNotes:                     "The first Bellatrix block post-Merge (with ExecutionPayload)",
			SignedBeaconBlock:             filepath.Join("ssz-data", "4700013", "signed-beacon-block.ssz"),
			BeaconState:                   filepath.Join("ssz-data", "4700013", "beacon-state.ssz"),
			CorrectEth1DataBlockHash:      "0xb8736ada384707e156f2e0e69d8311ceda11f96806921644a378fd55899894ca",
			CorrectParentRoot:             "0x60e751f7d2cf0ae24b195bda37e9add56a7d8c4b75469c018c0f912518c3bae8",
			CorrectSignedBeaconBlockMhKey: "/blocks/QLVAEQRQPA4DCMDBGAYDIMBQME4DAY3EMZTGGMJRMZTGIY3GGE3WCYZUGA2GCYZUMRRGCMRRGVRDSNJSGIYTSNJVME4WIZTEMRTDCNRTMQYGEMDE",
			CorrectBeaconStateMhKey:       "",
			CorrectExecutionPayloadHeader: &beaconclient.DbExecutionPayloadHeader{
				BlockNumber:      15537394,
				Timestamp:        1663224179,
				BlockHash:        "0x56a9bb0302da44b8c0b3df540781424684c3af04d0b7a38d72842b762076a664",
				ParentHash:       "0x55b11b918355b1ef9c5db810302ebad0bf2544255b530cdce90674d5887bb286",
				StateRoot:        "0x40c07091e16263270f3579385090fea02dd5f061ba6750228fcc082ff762fda7",
				ReceiptsRoot:     "0x928073fb98ce316265ea35d95ab7e2e1206cecd85242eb841dbbcc4f568fca4b",
				TransactionsRoot: "0xf9ef008aaf996dccd1c871c7e937f25d66e057e52773fbe2497090c114231acf",
			},
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
		bcUniqueIdentifier:      bcUniqueIdentifier,
		checkDb:                 true,
	}

	BeaconNodeTester = TestBeaconNode{
		TestEvents: TestEvents,
		TestConfig: TestConfig,
	}
)

type Message struct {
	HeadMessage                   beaconclient.Head                      // The head messsage that will be streamed to the BeaconClient
	TestNotes                     string                                 // A small explanation of the purpose this structure plays in the testing landscape.
	MimicConfig                   *MimicConfig                           // A configuration of parameters that you are trying to
	SignedBeaconBlock             string                                 // The file path output of an SSZ encoded SignedBeaconBlock.
	BeaconState                   string                                 // The file path output of an SSZ encoded BeaconState.
	CorrectSignedBeaconBlockMhKey string                                 // The correct MhKey for the signedBeaconBlock
	CorrectBeaconStateMhKey       string                                 // The correct MhKey beaconState
	CorrectParentRoot             string                                 // The correct parent root
	CorrectEth1DataBlockHash      string                                 // The correct eth1blockHash
	CorrectExecutionPayloadHeader *beaconclient.DbExecutionPayloadHeader // The correct ExecutionPayload details.
}

// A structure that can be utilized to mimic and existing SSZ object but change it ever so slightly.
// This is used because creating your own SSZ object is a headache.
type MimicConfig struct {
	ParentRoot  string // The parent root, leave it empty if you want a to use the universal
	ForkVersion string // Specify the fork version. This is needed as a workaround to create dummy SignedBeaconBlocks.
}

var _ = Describe("Capturehead", Label("head"), func() {

	Describe("Receiving New Head SSE messages", Label("unit", "behavioral"), func() {
		Context("Correctly formatted Phase0 Block", func() {
			It("Should turn it into a struct successfully.", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "99")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testProcessBlock(bc, BeaconNodeTester.TestEvents["100"].HeadMessage, 3, maxRetry, 1, 0, 0)
				if bc.PerformBeaconBlockProcessing {
					validateSignedBeaconBlock(bc, BeaconNodeTester.TestEvents["100"].HeadMessage, BeaconNodeTester.TestEvents["100"].CorrectParentRoot, BeaconNodeTester.TestEvents["100"].CorrectEth1DataBlockHash, BeaconNodeTester.TestEvents["100"].CorrectSignedBeaconBlockMhKey, BeaconNodeTester.TestEvents["100"].CorrectExecutionPayloadHeader)
				}
				if bc.PerformBeaconStateProcessing {
					validateBeaconState(bc, BeaconNodeTester.TestEvents["100"].HeadMessage, BeaconNodeTester.TestEvents["100"].CorrectBeaconStateMhKey)
				}

			})
		})
		Context("Correctly formatted Altair Block", func() {
			It("Should turn it into a struct successfully.", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "2375702")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testProcessBlock(bc, BeaconNodeTester.TestEvents["2375703"].HeadMessage, 74240, maxRetry, 1, 0, 0)
				if bc.PerformBeaconBlockProcessing {
					validateSignedBeaconBlock(bc, BeaconNodeTester.TestEvents["2375703"].HeadMessage, BeaconNodeTester.TestEvents["2375703"].CorrectParentRoot, BeaconNodeTester.TestEvents["2375703"].CorrectEth1DataBlockHash, BeaconNodeTester.TestEvents["2375703"].CorrectSignedBeaconBlockMhKey, BeaconNodeTester.TestEvents["2375703"].CorrectExecutionPayloadHeader)
				}
				if bc.PerformBeaconStateProcessing {
					validateBeaconState(bc, BeaconNodeTester.TestEvents["2375703"].HeadMessage, BeaconNodeTester.TestEvents["2375703"].CorrectBeaconStateMhKey)
				}
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
		Context("Correctly formatted Bellatrix Test Blocks", Label("unit", "bellatrix"), func() {
			It("Should turn it into a struct successfully (pre-Merge).", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "4636672")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testProcessBlock(bc, BeaconNodeTester.TestEvents["4636672"].HeadMessage, 144896, maxRetry, 1, 0, 0)
				if bc.PerformBeaconBlockProcessing {
					validateSignedBeaconBlock(bc, BeaconNodeTester.TestEvents["4636672"].HeadMessage, BeaconNodeTester.TestEvents["4636672"].CorrectParentRoot, BeaconNodeTester.TestEvents["4636672"].CorrectEth1DataBlockHash, BeaconNodeTester.TestEvents["4636672"].CorrectSignedBeaconBlockMhKey, BeaconNodeTester.TestEvents["4636672"].CorrectExecutionPayloadHeader)
				}
			})
			It("Should turn it into a struct successfully (post-Merge).", func() {
				bc := setUpTest(BeaconNodeTester.TestConfig, "4700013")
				BeaconNodeTester.SetupBeaconNodeMock(BeaconNodeTester.TestEvents, BeaconNodeTester.TestConfig.protocol, BeaconNodeTester.TestConfig.address, BeaconNodeTester.TestConfig.port, BeaconNodeTester.TestConfig.dummyParentRoot)
				defer httpmock.DeactivateAndReset()
				BeaconNodeTester.testProcessBlock(bc, BeaconNodeTester.TestEvents["4700013"].HeadMessage, 146875, maxRetry, 1, 0, 0)
				if bc.PerformBeaconBlockProcessing {
					validateSignedBeaconBlock(bc, BeaconNodeTester.TestEvents["4700013"].HeadMessage, BeaconNodeTester.TestEvents["4700013"].CorrectParentRoot, BeaconNodeTester.TestEvents["4700013"].CorrectEth1DataBlockHash, BeaconNodeTester.TestEvents["4700013"].CorrectSignedBeaconBlockMhKey, BeaconNodeTester.TestEvents["4700013"].CorrectExecutionPayloadHeader)
				}
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
			It("Should be able to get each objects root hash (100).", func() {
				testSszRoot(BeaconNodeTester.TestEvents["100"])
			})
		})
		Context("Altair: We have a correctly formated SSZ SignedBeaconBlock and BeaconState", func() {
			It("Should be able to get each objects root hash (2375703).", func() {
				testSszRoot(BeaconNodeTester.TestEvents["2375703"])
			})
		})
		//Context("When there is a skipped slot", func() {
		//	It("Should indicate that the slot was skipped", func() {

		//	})
		//})
		Context("When the proper SSZ objects are not served", func() {
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
		Context("Altair: Multiple reorgs have occurred on this slot", func() {
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
	bcUniqueIdentifier      int
	checkDb                 bool
}

//////////////////////////////////////////////////////
// Helper functions
//////////////////////////////////////////////////////

// Must run before each test. We can't use the beforeEach because of the way
// Gingko treats race conditions.
func setUpTest(config Config, maxSlot string) *beaconclient.BeaconClient {
	bc, err := beaconclient.CreateBeaconClient(context.Background(), config.protocol, config.address, config.port, config.knownGapsTableIncrement, config.bcUniqueIdentifier, config.checkDb)
	Expect(err).ToNot(HaveOccurred())
	db, err := postgres.SetupPostgresDb(config.dbHost, config.dbPort, config.dbName, config.dbUser, config.dbPassword, config.dbDriver)
	Expect(err).ToNot(HaveOccurred())

	// Drop all records from the DB.
	clearEthBeaconDbTables(db)

	// Add an slot to the eth_beacon.slots table so it we can control how known_gaps are handled.
	writeSlot(db, maxSlot)
	bc.Db = db

	return bc
}

// A helper function to validate the expected output from the eth_beacon.slots table.
func validateSlot(bc *beaconclient.BeaconClient, headMessage beaconclient.Head, correctEpoch uint64, correctStatus string) {
	epoch, dbSlot, blockRoot, stateRoot, status := queryDbSlotAndBlock(bc.Db, headMessage.Slot, headMessage.Block)
	log.Info("validateSlot: ", headMessage)
	baseSlot, err := strconv.ParseUint(headMessage.Slot, 10, 64)
	Expect(err).ToNot(HaveOccurred())
	Expect(dbSlot).To(Equal(baseSlot))
	Expect(epoch).To(Equal(correctEpoch))
	Expect(blockRoot).To(Equal(headMessage.Block))
	Expect(stateRoot).To(Equal(headMessage.State))
	Expect(status).To(Equal(correctStatus))
}

// A helper function to validate the expected output from the eth_beacon.signed_block table.
func validateSignedBeaconBlock(bc *beaconclient.BeaconClient, headMessage beaconclient.Head,
	correctParentRoot string, correctEth1DataBlockHash string, correctMhKey string,
	correctExecutionPayloadheader *beaconclient.DbExecutionPayloadHeader) {
	dbSignedBlock := queryDbSignedBeaconBlock(bc.Db, headMessage.Slot, headMessage.Block)
	log.Info("validateSignedBeaconBlock: ", headMessage)
	baseSlot, err := strconv.ParseUint(headMessage.Slot, 10, 64)
	Expect(err).ToNot(HaveOccurred())
	Expect(dbSignedBlock.Slot).To(Equal(baseSlot))
	Expect(dbSignedBlock.BlockRoot).To(Equal(headMessage.Block))
	Expect(dbSignedBlock.ParentBlock).To(Equal(correctParentRoot))
	Expect(dbSignedBlock.Eth1DataBlockHash).To(Equal(correctEth1DataBlockHash))
	Expect(dbSignedBlock.MhKey).To(Equal(correctMhKey))
	Expect(dbSignedBlock.ExecutionPayloadHeader).To(Equal(correctExecutionPayloadheader))
}

// A helper function to validate the expected output from the eth_beacon.state table.
func validateBeaconState(bc *beaconclient.BeaconClient, headMessage beaconclient.Head, correctMhKey string) {
	dbSlot, stateRoot, mhKey := queryDbBeaconState(bc.Db, headMessage.Slot, headMessage.State)
	log.Info("validateBeaconState: ", headMessage)
	baseSlot, err := strconv.Atoi(headMessage.Slot)
	Expect(err).ToNot(HaveOccurred())
	Expect(dbSlot).To(Equal(baseSlot))
	Expect(stateRoot).To(Equal(headMessage.State))
	Expect(mhKey).To(Equal(correctMhKey))

}

// Wrapper function to send a head message to the beaconclient
func sendHeadMessage(bc *beaconclient.BeaconClient, head beaconclient.Head, maxRetry int, expectedSuccessfulInserts uint64) {

	data, err := json.Marshal(head)
	Expect(err).ToNot(HaveOccurred())

	startInserts := atomic.LoadUint64(&bc.Metrics.SlotInserts)
	bc.HeadTracking.MessagesCh <- &sse.Event{
		ID:    []byte{},
		Data:  data,
		Event: []byte{},
		Retry: []byte{},
	}
	curRetry := 0
	for atomic.LoadUint64(&bc.Metrics.SlotInserts) != startInserts+expectedSuccessfulInserts {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			log.WithFields(log.Fields{
				"startInsert":               startInserts,
				"expectedSuccessfulInserts": expectedSuccessfulInserts,
				"currentValue":              atomic.LoadUint64(&bc.Metrics.SlotInserts),
			}).Error("HeadTracking Insert wasn't incremented properly.")
			Fail("Too many retries have occurred.")
		}
	}
}

// A helper function to query the eth_beacon.slots table based on the slot and block_root
func queryDbSlotAndBlock(db sql.Database, querySlot string, queryBlockRoot string) (uint64, uint64, string, string, string) {
	sqlStatement := `SELECT epoch, slot, block_root, state_root, status FROM eth_beacon.slots WHERE slot=$1 AND block_root=$2;`
	var epoch, slot uint64
	var blockRoot, stateRoot, status string
	log.Debug("Starting to query the eth_beacon.slots table, ", querySlot, " ", queryBlockRoot)
	err := db.QueryRow(context.Background(), sqlStatement, querySlot, queryBlockRoot).Scan(&epoch, &slot, &blockRoot, &stateRoot, &status)
	Expect(err).ToNot(HaveOccurred())
	log.Debug("Querying the eth_beacon.slots table complete")
	return epoch, slot, blockRoot, stateRoot, status
}

// A helper function to query the eth_beacon.signed_block table based on the slot and block_root.
func queryDbSignedBeaconBlock(db sql.Database, querySlot string, queryBlockRoot string) beaconclient.DbSignedBeaconBlock {
	sqlStatement := `SELECT slot, block_root, parent_block_root, eth1_data_block_hash, mh_key, 
       payload_block_number, payload_timestamp, payload_block_hash,
       payload_parent_hash, payload_state_root, payload_receipts_root,
       payload_transactions_root FROM eth_beacon.signed_block WHERE slot=$1 AND block_root=$2;`

	var slot uint64
	var payloadBlockNumber, payloadTimestamp *uint64
	var blockRoot, parentBlockRoot, eth1DataBlockHash, mhKey string
	var payloadBlockHash, payloadParentHash, payloadStateRoot, payloadReceiptsRoot, payloadTransactionsRoot *string

	row := db.QueryRow(context.Background(), sqlStatement, querySlot, queryBlockRoot)
	err := row.Scan(&slot, &blockRoot, &parentBlockRoot, &eth1DataBlockHash, &mhKey,
		&payloadBlockNumber, &payloadTimestamp, &payloadBlockHash,
		&payloadParentHash, &payloadStateRoot, &payloadReceiptsRoot, &payloadTransactionsRoot)
	Expect(err).ToNot(HaveOccurred())

	signedBlock := beaconclient.DbSignedBeaconBlock{
		Slot:                   slot,
		BlockRoot:              blockRoot,
		ParentBlock:            parentBlockRoot,
		Eth1DataBlockHash:      eth1DataBlockHash,
		MhKey:                  mhKey,
		ExecutionPayloadHeader: nil,
	}

	if nil != payloadBlockNumber {
		signedBlock.ExecutionPayloadHeader = &beaconclient.DbExecutionPayloadHeader{
			BlockNumber:      *payloadBlockNumber,
			Timestamp:        *payloadTimestamp,
			BlockHash:        *payloadBlockHash,
			ParentHash:       *payloadParentHash,
			StateRoot:        *payloadStateRoot,
			ReceiptsRoot:     *payloadReceiptsRoot,
			TransactionsRoot: *payloadTransactionsRoot,
		}
	}

	return signedBlock
}

// A helper function to query the eth_beacon.signed_block table based on the slot and block_root.
func queryDbBeaconState(db sql.Database, querySlot string, queryStateRoot string) (uint64, string, string) {
	sqlStatement := `SELECT slot, state_root, mh_key FROM eth_beacon.state WHERE slot=$1 AND state_root=$2;`
	var slot uint64
	var stateRoot, mhKey string
	row := db.QueryRow(context.Background(), sqlStatement, querySlot, queryStateRoot)
	err := row.Scan(&slot, &stateRoot, &mhKey)
	Expect(err).ToNot(HaveOccurred())
	return slot, stateRoot, mhKey
}

// Count the entries in the knownGaps table.
func countKnownGapsTable(db sql.Database) int {
	var count int
	sqlStatement := "SELECT COUNT(*) FROM eth_beacon.known_gaps"
	err := db.QueryRow(context.Background(), sqlStatement).Scan(&count)
	Expect(err).ToNot(HaveOccurred())
	return count
}

// Return the start and end slot
func queryKnownGaps(db sql.Database, queryStartGap string, QueryEndGap string) (int, int) {
	sqlStatement := `SELECT start_slot, end_slot FROM eth_beacon.known_gaps WHERE start_slot=$1 AND end_slot=$2;`
	var startGap, endGap int
	row := db.QueryRow(context.Background(), sqlStatement, queryStartGap, QueryEndGap)
	err := row.Scan(&startGap, &endGap)
	Expect(err).ToNot(HaveOccurred())
	return startGap, endGap
}

// A function that will remove all entries from the eth_beacon tables for you.
func clearEthBeaconDbTables(db sql.Database) {
	deleteQueries := []string{"DELETE FROM eth_beacon.slots;", "DELETE FROM eth_beacon.signed_block;", "DELETE FROM eth_beacon.state;", "DELETE FROM eth_beacon.known_gaps;", "DELETE FROM eth_beacon.historic_process;", "DELETE FROM public.blocks;"}
	for _, queries := range deleteQueries {
		_, err := db.Exec(context.Background(), queries)
		Expect(err).ToNot(HaveOccurred())
	}
}

// Write an entry to the eth_beacon.slots table with just a slot number
func writeSlot(db sql.Database, slot string) {
	_, err := db.Exec(context.Background(), beaconclient.UpsertSlotsStmt, "0", slot, "", "", "")
	Expect(err).ToNot(HaveOccurred())
}

// Read a file with the SignedBeaconBlock in SSZ and return the SSZ object. This is used for testing only.
// We can't use the readSignedBeaconBlockInterface to update struct fields so this is the workaround.
func readSignedBeaconBlock(slotFile string) (*beaconclient.SignedBeaconBlock, error) {
	dat, err := os.ReadFile(slotFile)
	if err != nil {
		return nil, fmt.Errorf("Can't find the slot file, %s", slotFile)
	}
	var block beaconclient.SignedBeaconBlock
	err = block.UnmarshalSSZ(dat)
	Expect(err).ToNot(HaveOccurred())
	return &block, nil
}

// Read a file with the BeaconState in SSZ and return the SSZ object
func readBeaconState(slotFile string) (*beaconclient.BeaconState, error) {
	dat, err := os.ReadFile(slotFile)
	if err != nil {
		return nil, fmt.Errorf("Can't find the slot file, %s", slotFile)
	}
	var beaconState beaconclient.BeaconState
	err = beaconState.UnmarshalSSZ(dat)
	Expect(err).ToNot(HaveOccurred())
	return &beaconState, nil
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
				return httpmock.NewStringResponse(404, fmt.Sprintf("Unable to find file for %s", id)), nil
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
				return httpmock.NewStringResponse(404, fmt.Sprintf("Unable to find file for %s", id)), nil
			}
			return httpmock.NewBytesResponse(200, dat), nil
		},
	)
	// Not needed but could be useful to have.
	blockRootUrl := `=~^` + protocol + "://" + address + ":" + strconv.Itoa(port) + "/eth/v1/beacon/blocks/" + `([^/]+)` + "/root"
	httpmock.RegisterResponder("GET", blockRootUrl,
		func(req *http.Request) (*http.Response, error) {
			// Get ID from request
			slot := httpmock.MustGetSubmatch(req, 1)
			dat, err := tbc.provideBlockRoot(slot)
			if err != nil {
				Expect(err).NotTo(HaveOccurred())
				return httpmock.NewStringResponse(404, fmt.Sprintf("Unable to find block root for %s", slot)), err
			}
			return httpmock.NewBytesResponse(200, dat), nil
		},
	)
}

// Provide the Block root
func (tbc TestBeaconNode) provideBlockRoot(slot string) ([]byte, error) {
	for _, val := range tbc.TestEvents {
		if val.HeadMessage.Slot == slot && val.MimicConfig == nil {
			block, err := hex.DecodeString(val.HeadMessage.Block[2:])
			Expect(err).ToNot(HaveOccurred())
			return block, nil
		}
	}
	return nil, fmt.Errorf("Unable to find the Blockroot in test object.")
}

// A function to mimic querying the state from the beacon node. We simply get the SSZ file are return it.
func (tbc TestBeaconNode) provideSsz(slotIdentifier string, sszIdentifier string, dummyParentRoot string) ([]byte, error) {
	var slotFile string
	var Message Message

	for _, val := range tbc.TestEvents {
		if sszIdentifier == "state" {
			if (val.HeadMessage.Slot == slotIdentifier && val.MimicConfig == nil) || val.HeadMessage.State == slotIdentifier {
				slotFile = val.BeaconState
				Message = val
			}
		} else if sszIdentifier == "block" {
			if (val.HeadMessage.Slot == slotIdentifier && val.MimicConfig == nil) || val.HeadMessage.Block == slotIdentifier {
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
				Expect(block.IsPhase0()).To(BeTrue())
				var phase0 = block.GetPhase0()

				slot, err := strconv.ParseUint(Message.HeadMessage.Slot, 10, 64)
				Expect(err).ToNot(HaveOccurred())
				phase0.Message.Slot = common.Slot(slot)

				phase0.Message.StateRoot, err = decodeRoot(Message.HeadMessage.State)
				Expect(err).ToNot(HaveOccurred())

				if Message.MimicConfig.ParentRoot == "" {
					phase0.Message.ParentRoot, err = decodeRoot(dummyParentRoot)
					Expect(err).ToNot(HaveOccurred())
				} else {
					phase0.Message.ParentRoot, err = decodeRoot(Message.MimicConfig.ParentRoot)
					Expect(err).ToNot(HaveOccurred())
				}
				return block.MarshalSSZ()
			case "altair":
				block, err := readSignedBeaconBlock(slotFile)
				if err != nil {
					return nil, err
				}
				Expect(block.IsAltair()).To(BeTrue())
				var altair = block.GetAltair()
				slot, err := strconv.ParseUint(Message.HeadMessage.Slot, 10, 64)
				Expect(err).ToNot(HaveOccurred())
				altair.Message.Slot = common.Slot(slot)

				altair.Message.StateRoot, err = decodeRoot(Message.HeadMessage.State)
				Expect(err).ToNot(HaveOccurred())

				if Message.MimicConfig.ParentRoot == "" {
					altair.Message.ParentRoot, err = decodeRoot(dummyParentRoot)
					Expect(err).ToNot(HaveOccurred())
				} else {
					altair.Message.ParentRoot, err = decodeRoot(Message.MimicConfig.ParentRoot)
					Expect(err).ToNot(HaveOccurred())
				}
				return block.MarshalSSZ()
			}
		}
		if sszIdentifier == "state" {
			state, err := readBeaconState(slotFile)
			if err != nil {
				return nil, err
			}
			slot, err := strconv.ParseUint(Message.HeadMessage.Slot, 10, 64)
			Expect(err).ToNot(HaveOccurred())
			if state.IsBellatrix() {
				state.GetBellatrix().Slot = common.Slot(slot)
			} else if state.IsAltair() {
				state.GetAltair().Slot = common.Slot(slot)
			} else {
				state.GetPhase0().Slot = common.Slot(slot)
			}
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
func (tbc TestBeaconNode) testMultipleReorgs(bc *beaconclient.BeaconClient, firstHead beaconclient.Head, secondHead beaconclient.Head, thirdHead beaconclient.Head, epoch uint64, maxRetry int) {
	go bc.CaptureHead()
	time.Sleep(1 * time.Second)

	log.Info("Sending Messages to BeaconClient")
	sendHeadMessage(bc, firstHead, maxRetry, 1)
	sendHeadMessage(bc, secondHead, maxRetry, 1)
	sendHeadMessage(bc, thirdHead, maxRetry, 1)

	curRetry := 0
	for atomic.LoadUint64(&bc.Metrics.ReorgInserts) != 2 {
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
		Epoch:               strconv.FormatUint(epoch, 10),
		ExecutionOptimistic: false,
	})
	Expect(err).ToNot(HaveOccurred())
	bc.ReOrgTracking.MessagesCh <- &sse.Event{
		Data: data,
	}

	curRetry = 0
	for atomic.LoadUint64(&bc.Metrics.ReorgInserts) != 3 {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			Fail("Too many retries have occurred.")
		}
	}

	if bc.Metrics.KnownGapsInserts != 0 {
		Fail("We found gaps when processing a single block")
	}

	log.Info("Make sure the forks were properly updated!")

	validateSlot(bc, firstHead, epoch, "forked")
	validateSlot(bc, secondHead, epoch, "proposed")
	validateSlot(bc, thirdHead, epoch, "forked")

}

// A test to validate a single block was processed correctly
func (tbc TestBeaconNode) testProcessBlock(bc *beaconclient.BeaconClient, head beaconclient.Head, epoch uint64, maxRetry int, expectedSuccessInsert uint64, expectedKnownGaps uint64, expectedReorgs uint64) {
	go bc.CaptureHead()
	time.Sleep(1 * time.Second)
	sendHeadMessage(bc, head, maxRetry, expectedSuccessInsert)

	curRetry := 0
	for atomic.LoadUint64(&bc.Metrics.KnownGapsInserts) != expectedKnownGaps {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			Fail(fmt.Sprintf("Wrong gap metrics, got: %d, wanted %d", bc.Metrics.KnownGapsInserts, expectedKnownGaps))
		}
	}

	curRetry = 0
	for atomic.LoadUint64(&bc.Metrics.ReorgInserts) != expectedReorgs {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			Fail(fmt.Sprintf("Wrong reorg metrics, got: %d, wanted %d", bc.Metrics.KnownGapsInserts, expectedKnownGaps))
		}
	}

	if expectedSuccessInsert > 0 {
		validateSlot(bc, head, epoch, "proposed")
	}
}

// A test that ensures that if two HeadMessages occur for a single slot they are marked
// as proposed and forked correctly.
func (tbc TestBeaconNode) testMultipleHead(bc *beaconclient.BeaconClient, firstHead beaconclient.Head, secondHead beaconclient.Head, epoch uint64, maxRetry int) {
	go bc.CaptureHead()
	time.Sleep(1 * time.Second)

	sendHeadMessage(bc, firstHead, maxRetry, 1)
	sendHeadMessage(bc, secondHead, maxRetry, 1)

	curRetry := 0
	for atomic.LoadUint64(&bc.Metrics.ReorgInserts) != 1 {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			Fail(" Too many retries have occurred.")
		}
	}

	if bc.Metrics.KnownGapsInserts != 0 {
		Fail("We found gaps when processing a single block")
	}

	log.Info("Checking Altair to make sure the fork was marked properly.")
	validateSlot(bc, firstHead, epoch, "forked")
	validateSlot(bc, secondHead, epoch, "proposed")
}

// A test that ensures that if two HeadMessages occur for a single slot they are marked
// as proposed and forked correctly.
func (tbc TestBeaconNode) testKnownGapsMessages(bc *beaconclient.BeaconClient, tableIncrement int, expectedEntries uint64, maxRetry int, msg ...beaconclient.Head) {
	bc.KnownGapTableIncrement = tableIncrement
	go bc.CaptureHead()
	time.Sleep(1 * time.Second)

	for _, headMsg := range msg {
		sendHeadMessage(bc, headMsg, maxRetry, 1)
	}

	curRetry := 0
	for atomic.LoadUint64(&bc.Metrics.KnownGapsInserts) != expectedEntries {
		time.Sleep(1 * time.Second)
		curRetry = curRetry + 1
		if curRetry == maxRetry {
			Fail("Too many retries have occurred.")
		}
	}

	log.Info("Checking to make sure we have the expected number of entries in the knownGaps table.")
	knownGapCount := countKnownGapsTable(bc.Db)
	Expect(knownGapCount).To(Equal(int(expectedEntries)))

	if atomic.LoadUint64(&bc.Metrics.ReorgInserts) != 0 {
		Fail("We found reorgs when we didn't expect it")
	}
}

// This function will make sure we are properly able to get the SszRoot of the SignedBeaconBlock and the BeaconState.
func testSszRoot(msg Message) {
	state, err := readBeaconState(msg.BeaconState)
	Expect(err).ToNot(HaveOccurred())
	stateRoot := state.HashTreeRoot()
	Expect(err).ToNot(HaveOccurred())
	Expect(msg.HeadMessage.State).To(Equal("0x" + hex.EncodeToString(stateRoot[:])))

	block, err := readSignedBeaconBlock(msg.SignedBeaconBlock)
	Expect(err).ToNot(HaveOccurred())
	blockRoot := block.Block().HashTreeRoot()
	Expect(err).ToNot(HaveOccurred())
	Expect(msg.HeadMessage.Block).To(Equal("0x" + hex.EncodeToString(blockRoot[:])))
}

func decodeRoot(raw string) (common.Root, error) {
	value, err := hex.DecodeString(raw)
	if err != nil {
		return common.Root{}, err
	}
	var root common.Root
	copy(root[:], value[:32])
	return root, nil
}
