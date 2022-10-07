package beaconclient_test

import (
	"context"
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	zcommon "github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/tree"
	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/beaconclient"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/database/sql"
	"math/big"
	"time"
)

var _ = Describe("e2emerge", Label("e2e"), func() {
	e2eConfig := TestConfig
	e2eConfig.port = 5052
	e2eConfig.performBeaconStateProcessing = false
	e2eConfig.performBeaconBlockProcessing = true

	level, _ := log.ParseLevel("debug")
	log.SetLevel(level)

	Describe("Run the application against a Merge testnet", func() {
		Context("When we send a TX to geth", func() {
			It("We should see the TX included in the ExecutionPayload of a BeaconBlock", func() {
				bc := setUpTest(e2eConfig, "0")
				go bc.CaptureHead()

				tx, _ := sendTestTx()

				beaconBlock := waitForTxToBeIndexed(bc.Db, tx)
				Expect(beaconBlock).ToNot(BeNil())
			})
		})
	})
})

type SentTx struct {
	hash      string
	raw       []byte
	blockNo   uint64
	blockHash string
	txIndex   uint
}

func (tx *SentTx) RawHex() string {
	return "0x" + hex.EncodeToString(tx.raw)
}

func waitForTxToBeIndexed(db sql.Database, tx *SentTx) *beaconclient.DbSignedBeaconBlock {
	var beaconBlock *beaconclient.DbSignedBeaconBlock = nil
	for i := 0; i < 30; i++ {
		time.Sleep(time.Second)
		record := lookForTxInDb(db, tx)
		if nil != record {
			beaconBlock = record
			log.Debugf("Found ETH1 TX %s in SignedBeaconBlock %d/%s", tx.hash, beaconBlock.Slot, beaconBlock.BlockRoot)
			break
		}
	}
	return beaconBlock
}

func lookForTxInDb(db sql.Database, tx *SentTx) *beaconclient.DbSignedBeaconBlock {
	sqlStatement := `SELECT * FROM eth_beacon.signed_block WHERE 
                                    payload_block_number = $1 AND
                                    payload_block_hash = $2 AND
                                    payload_transactions_root = $3`

	// We can pre-calculate the root and query on it because we are only sending a single TX at a time.
	// Otherwise we would need to lookup the root by block num+hash, then do a proof that its txs
	// root includes our TX.
	var ptxs = zcommon.PayloadTransactions{tx.raw}
	txRoot := ptxs.HashTreeRoot(configs.Mainnet, tree.GetHashFn())

	var slot uint64
	var blockRoot, parentBlock, eth1DataBlockHash, mhKey string

	var blockNumber, timestamp uint64
	var blockHash, parentHash, stateRoot, receiptsRoot, transactionsRoot string

	err := db.
		QueryRow(context.Background(), sqlStatement, tx.blockNo, tx.blockHash, "0x"+hex.EncodeToString(txRoot[:])).
		Scan(&slot, &blockRoot, &parentBlock, &eth1DataBlockHash, &mhKey,
			&blockNumber, &timestamp, &blockHash, &parentHash, &stateRoot,
			&receiptsRoot, &transactionsRoot)
	if nil != err {
		return nil
	}

	return &beaconclient.DbSignedBeaconBlock{
		Slot:              slot,
		BlockRoot:         blockRoot,
		ParentBlock:       parentBlock,
		Eth1DataBlockHash: eth1DataBlockHash,
		MhKey:             mhKey,
		ExecutionPayloadHeader: &beaconclient.DbExecutionPayloadHeader{
			BlockNumber:      blockNumber,
			Timestamp:        timestamp,
			BlockHash:        blockHash,
			ParentHash:       parentHash,
			StateRoot:        stateRoot,
			ReceiptsRoot:     receiptsRoot,
			TransactionsRoot: transactionsRoot,
		},
	}
}

func sendTestTx() (*SentTx, error) {
	ctx := context.Background()
	eth, err := createClient()
	Expect(err).ToNot(HaveOccurred())

	tx, err := sendTransaction(
		ctx,
		eth,
		"0xe6ce22afe802caf5ff7d3845cec8c736ecc8d61f",
		"0xe22AD83A0dE117bA0d03d5E94Eb4E0d80a69C62a",
		10,
		"0x888814df89c4358d7ddb3fa4b0213e7331239a80e1f013eaa7b2deca2a41a218",
	)
	Expect(err).ToNot(HaveOccurred())

	txBin, err := tx.MarshalBinary()
	Expect(err).ToNot(HaveOccurred())

	for i := 0; i <= 30; i++ {
		time.Sleep(time.Second)
		receipt, _ := eth.TransactionReceipt(ctx, tx.Hash())
		if nil != receipt {
			sentTx := &SentTx{
				hash:      tx.Hash().String(),
				raw:       txBin,
				blockNo:   receipt.BlockNumber.Uint64(),
				blockHash: receipt.BlockHash.String(),
				txIndex:   receipt.TransactionIndex,
			}
			log.Debugf("Sent ETH1 TX %s (Block No: %d, Block Hash: %s)", sentTx.hash, sentTx.blockNo, sentTx.blockHash)
			return sentTx, nil
		}
	}

	err = errors.New("Timed out waiting for TX.")
	Expect(err).ToNot(HaveOccurred())
	return nil, err
}

func createClient() (*ethclient.Client, error) {
	return ethclient.Dial("http://localhost:8545")
}

// sendTransaction sends a transaction with 1 ETH to a specified address.
func sendTransaction(ctx context.Context, eth *ethclient.Client, fromAddr string, toAddr string, amount int64, signingKey string) (*types.Transaction, error) {
	var (
		from     = common.HexToAddress(fromAddr)
		to       = common.HexToAddress(toAddr)
		sk       = crypto.ToECDSAUnsafe(common.FromHex(signingKey))
		value    = big.NewInt(amount)
		gasLimit = uint64(21000)
	)
	// Retrieve the chainid (needed for signer)
	chainid, err := eth.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	// Retrieve the pending nonce
	nonce, err := eth.PendingNonceAt(ctx, from)
	if err != nil {
		return nil, err
	}
	// Get suggested gas price
	tipCap, _ := eth.SuggestGasTipCap(ctx)
	feeCap, _ := eth.SuggestGasPrice(ctx)
	// Create a new transaction
	tx := types.NewTx(
		&types.DynamicFeeTx{
			ChainID:   chainid,
			Nonce:     nonce,
			GasTipCap: tipCap,
			GasFeeCap: feeCap,
			Gas:       gasLimit,
			To:        &to,
			Value:     value,
			Data:      nil,
		})
	// Sign the transaction using our keys
	signedTx, _ := types.SignTx(tx, types.NewLondonSigner(chainid), sk)
	// Send the transaction to our node
	return signedTx, eth.SendTransaction(ctx, signedTx)
}
