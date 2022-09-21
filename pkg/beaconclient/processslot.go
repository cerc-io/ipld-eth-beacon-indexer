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
// This file will keep track of all the code needed to process a slot.
// To process a slot, it should have all the necessary data needed to write it to the DB.
// But not actually write it.

package beaconclient

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/loghelper"
	"golang.org/x/sync/errgroup"
)

type SlotProcessingDetails struct {
	Context                      context.Context      // A context generic context with multiple uses.
	ServerEndpoint               string               // What is the endpoint of the beacon server.
	Db                           sql.Database         // Database object used for reads and writes.
	Metrics                      *BeaconClientMetrics // An object used to keep track of certain BeaconClient Metrics.
	KnownGapTableIncrement       int                  // The max number of slots within a single known_gaps table entry.
	CheckDb                      bool                 // Should we check the DB to see if the slot exists before processing it?
	PerformBeaconStateProcessing bool                 // Should we process BeaconStates?
	PerformBeaconBlockProcessing bool                 // Should we process BeaconBlocks?

	StartingSlot      uint64 // If we're performing head tracking. What is the first slot we processed.
	PreviousSlot      uint64 // Whats the previous slot we processed
	PreviousBlockRoot string // Whats the previous block root, used to check the next blocks parent.
}

func (bc *BeaconClient) SlotProcessingDetails() SlotProcessingDetails {
	return SlotProcessingDetails{
		Context:        bc.Context,
		ServerEndpoint: bc.ServerEndpoint,
		Db:             bc.Db,
		Metrics:        bc.Metrics,

		CheckDb:                      bc.CheckDb,
		PerformBeaconBlockProcessing: bc.PerformBeaconBlockProcessing,
		PerformBeaconStateProcessing: bc.PerformBeaconStateProcessing,

		KnownGapTableIncrement: bc.KnownGapTableIncrement,
		StartingSlot:           bc.StartingSlot,
		PreviousSlot:           bc.PreviousSlot,
		PreviousBlockRoot:      bc.PreviousBlockRoot,
	}
}

type ProcessSlot struct {
	// Generic

	Slot               uint64               // The slot number.
	Epoch              uint64               // The epoch number.
	BlockRoot          string               // The hex encoded string of the BlockRoot.
	StateRoot          string               // The hex encoded string of the StateRoot.
	ParentBlockRoot    string               // The hex encoded string of the parent block.
	Status             string               // The status of the block
	HeadOrHistoric     string               // Is this the head or a historic slot. This is critical when trying to analyze errors and skipped slots.
	Db                 sql.Database         // The DB object used to write to the DB.
	Metrics            *BeaconClientMetrics // An object to keep track of the beaconclient metrics
	PerformanceMetrics PerformanceMetrics   // An object to keep track of performance metrics.
	// BeaconBlock

	SszSignedBeaconBlock  []byte             // The entire SSZ encoded SignedBeaconBlock
	FullSignedBeaconBlock *SignedBeaconBlock // The unmarshaled BeaconState object, the unmarshalling could have errors.

	// BeaconState
	FullBeaconState *BeaconState // The unmarshaled BeaconState object, the unmarshalling could have errors.
	SszBeaconState  []byte       // The entire SSZ encoded BeaconState

	// DB Write objects
	DbSlotsModel             *DbSlots             // The model being written to the slots table.
	DbSignedBeaconBlockModel *DbSignedBeaconBlock // The model being written to the signed_block table.
	DbBeaconState            *DbBeaconState       // The model being written to the state table.
}

type PerformanceMetrics struct {
	BeaconNodeBlockRetrievalTime time.Duration // How long it took to get the BeaconBlock from the Beacon Node.
	BeaconNodeStateRetrievalTime time.Duration // How long it took to get the BeaconState from the Beacon Node.
	ParseBeaconObjectForHash     time.Duration // How long it took to get some information from the beacon objects.
	CheckDbPreProcessing         time.Duration // How long it takes to check the DB before processing a block.
	CreateDbWriteObject          time.Duration // How long it takes to create a DB write object.
	TransactSlotOnly             time.Duration // How long it takes to transact the slot information only.
	CheckReorg                   time.Duration // How long it takes to check for Reorgs
	CommitTransaction            time.Duration // How long it takes to commit the final transaction.
	TotalDbTransaction           time.Duration // How long it takes from start to committing the entire DB transaction.
	TotalProcessing              time.Duration // How long it took to process the entire slot.
}

// This function will do all the work to process the slot and write it to the DB.
// It will return the error and error process. The error process is used for providing reach detail to the
// known_gaps table.
func processFullSlot(
	ctx context.Context,
	slot uint64,
	blockRoot string,
	stateRoot string,
	previousSlot uint64,
	previousBlockRoot string,
	knownGapsTableIncrement int,
	headOrHistoric string,
	spd *SlotProcessingDetails) (error, string) {
	select {
	case <-ctx.Done():
		return nil, ""
	default:
		totalStart := time.Now()
		ps := &ProcessSlot{
			Slot:           slot,
			BlockRoot:      blockRoot,
			StateRoot:      stateRoot,
			HeadOrHistoric: headOrHistoric,
			Db:             spd.Db,
			Metrics:        spd.Metrics,
			PerformanceMetrics: PerformanceMetrics{
				BeaconNodeBlockRetrievalTime: 0,
				BeaconNodeStateRetrievalTime: 0,
				ParseBeaconObjectForHash:     0,
				CheckDbPreProcessing:         0,
				CreateDbWriteObject:          0,
				TransactSlotOnly:             0,
				CheckReorg:                   0,
				CommitTransaction:            0,
				TotalDbTransaction:           0,
				TotalProcessing:              0,
			},
		}

		g, _ := errgroup.WithContext(context.Background())

		if spd.PerformBeaconStateProcessing {
			// Get the BeaconState.
			g.Go(func() error {
				select {
				case <-ctx.Done():
					return nil
				default:
					start := time.Now()
					err := ps.getBeaconState(spd.ServerEndpoint)
					if err != nil {
						return err
					}
					ps.PerformanceMetrics.BeaconNodeStateRetrievalTime = time.Since(start)
					return nil
				}
			})
		}

		if spd.PerformBeaconBlockProcessing {
			// Get the SignedBeaconBlock.
			g.Go(func() error {
				select {
				case <-ctx.Done():
					return nil
				default:
					start := time.Now()
					err := ps.getSignedBeaconBlock(spd.ServerEndpoint)
					if err != nil {
						return err
					}
					ps.PerformanceMetrics.BeaconNodeBlockRetrievalTime = time.Since(start)
					return nil
				}
			})
		}

		if err := g.Wait(); err != nil {
			return err, "processSlot"
		}

		parseBeaconTime := time.Now()
		finalBlockRoot, finalStateRoot, _, err := ps.provideFinalHash()
		if err != nil {
			return err, "CalculateBlockRoot"
		}
		ps.PerformanceMetrics.ParseBeaconObjectForHash = time.Since(parseBeaconTime)

		if spd.CheckDb {
			checkDbTime := time.Now()
			var blockRequired bool
			if spd.PerformBeaconBlockProcessing {
				blockExists, err := checkSlotAndRoot(ps.Db, CheckSignedBeaconBlockStmt, strconv.FormatUint(ps.Slot, 10), finalBlockRoot)
				if err != nil {
					return err, "checkDb"
				}
				blockRequired = !blockExists
			}

			var stateRequired bool
			if spd.PerformBeaconStateProcessing {
				stateExists, err := checkSlotAndRoot(ps.Db, CheckBeaconStateStmt, strconv.FormatUint(ps.Slot, 10), finalStateRoot)
				if err != nil {
					return err, "checkDb"
				}
				stateRequired = !stateExists
			}

			if !blockRequired && !stateRequired {
				log.WithField("slot", slot).Info("Slot already in the DB.")
				return nil, ""
			}
			ps.PerformanceMetrics.CheckDbPreProcessing = time.Since(checkDbTime)
		}

		// Get this object ready to write
		createDbWriteTime := time.Now()
		dw, err := ps.createWriteObjects()
		if err != nil {
			return err, "blockRoot"
		}
		ps.PerformanceMetrics.CreateDbWriteObject = time.Since(createDbWriteTime)

		// Write the object to the DB.
		dbFullTransactionTime := time.Now()
		defer func() {
			err := dw.Tx.Rollback(dw.Ctx)
			if err != nil && err != pgx.ErrTxClosed {
				loghelper.LogError(err).Error("We were unable to Rollback a transaction")
			}
		}()

		transactionTime := time.Now()
		err = dw.transactFullSlot()
		if err != nil {
			return err, "processSlot"
		}
		ps.PerformanceMetrics.TransactSlotOnly = time.Since(transactionTime)

		// Handle any reorgs or skipped slots.
		reorgTime := time.Now()
		headOrHistoric = strings.ToLower(headOrHistoric)
		if headOrHistoric != "head" && headOrHistoric != "historic" {
			return fmt.Errorf("headOrHistoric must be either historic or head"), ""
		}
		if ps.HeadOrHistoric == "head" && previousSlot != 0 && previousBlockRoot != "" && ps.Status != "skipped" {
			ps.checkPreviousSlot(dw.Tx, dw.Ctx, previousSlot, previousBlockRoot, knownGapsTableIncrement)
		}
		ps.PerformanceMetrics.CheckReorg = time.Since(reorgTime)

		// Commit the transaction
		commitTime := time.Now()
		if err = dw.Tx.Commit(dw.Ctx); err != nil {
			return err, "transactionCommit"
		}
		ps.PerformanceMetrics.CommitTransaction = time.Since(commitTime)

		// Total metric capture time.
		ps.PerformanceMetrics.TotalDbTransaction = time.Since(dbFullTransactionTime)
		ps.PerformanceMetrics.TotalProcessing = time.Since(totalStart)

		log.WithFields(log.Fields{
			"slot":               slot,
			"performanceMetrics": fmt.Sprintf("%+v\n", ps.PerformanceMetrics),
		}).Debug("Performance Metric output!")

		return nil, ""
	}
}

// Handle a slot that is at head. A wrapper function for calling `handleFullSlot`.
func processHeadSlot(slot uint64, blockRoot string, stateRoot string, spd SlotProcessingDetails) {
	// Get the knownGaps at startUp
	if spd.PreviousSlot == 0 && spd.PreviousBlockRoot == "" {
		writeStartUpGaps(spd.Db, spd.KnownGapTableIncrement, slot, spd.Metrics)
	}
	// TODO(telackey): Why context.Background()?
	err, errReason := processFullSlot(context.Background(), slot, blockRoot, stateRoot,
		spd.PreviousSlot, spd.PreviousBlockRoot, spd.KnownGapTableIncrement, "head", &spd)
	if err != nil {
		writeKnownGaps(spd.Db, spd.KnownGapTableIncrement, slot, slot, err, errReason, spd.Metrics)
	}
}

// Handle a historic slot. A wrapper function for calling `handleFullSlot`.
func handleHistoricSlot(ctx context.Context, slot uint64, spd SlotProcessingDetails) (error, string) {
	return processFullSlot(ctx, slot, "", "", 0, "",
		1, "historic", &spd)
}

// Update the SszSignedBeaconBlock and FullSignedBeaconBlock object with their respective values.
func (ps *ProcessSlot) getSignedBeaconBlock(serverAddress string) error {
	var blockIdentifier string // Used to query the block
	if ps.BlockRoot != "" {
		blockIdentifier = ps.BlockRoot
	} else {
		blockIdentifier = strconv.FormatUint(ps.Slot, 10)
	}

	blockEndpoint := serverAddress + BcBlockQueryEndpoint + blockIdentifier
	sszSignedBeaconBlock, rc, err := querySsz(blockEndpoint, strconv.FormatUint(ps.Slot, 10))

	if err != nil || rc != 200 {
		loghelper.LogSlotError(strconv.FormatUint(ps.Slot, 10), err).Error("Unable to properly query the slot.")
		ps.FullSignedBeaconBlock = nil
		ps.SszSignedBeaconBlock = []byte{}
		ps.ParentBlockRoot = ""
		ps.Status = "skipped"

		// A 404 is normal in the case of a "skipped" slot.
		if rc == 404 {
			return nil
		}
		return err
	}

	var signedBeaconBlock SignedBeaconBlock
	err = signedBeaconBlock.UnmarshalSSZ(sszSignedBeaconBlock)
	if err != nil {
		loghelper.LogSlotError(strconv.FormatUint(ps.Slot, 10), err).Error("Unable to unmarshal SignedBeaconBlock for slot.")
		ps.FullSignedBeaconBlock = nil
		ps.SszSignedBeaconBlock = []byte{}
		ps.ParentBlockRoot = ""
		ps.Status = "skipped"
		return err
	}

	ps.FullSignedBeaconBlock = &signedBeaconBlock
	ps.SszSignedBeaconBlock = sszSignedBeaconBlock

	ps.ParentBlockRoot = toHex(ps.FullSignedBeaconBlock.Block().ParentRoot())
	return nil
}

// Update the SszBeaconState and FullBeaconState object with their respective values.
func (ps *ProcessSlot) getBeaconState(serverEndpoint string) error {
	var stateIdentifier string // Used to query the state
	if ps.StateRoot != "" {
		stateIdentifier = ps.StateRoot
	} else {
		stateIdentifier = strconv.FormatUint(ps.Slot, 10)
	}

	stateEndpoint := serverEndpoint + BcStateQueryEndpoint + stateIdentifier
	sszBeaconState, _, err := querySsz(stateEndpoint, strconv.FormatUint(ps.Slot, 10))
	if err != nil {
		loghelper.LogSlotError(strconv.FormatUint(ps.Slot, 10), err).Error("Unable to properly query the BeaconState.")
		return err
	}

	var beaconState BeaconState
	err = beaconState.UnmarshalSSZ(sszBeaconState)
	if err != nil {
		loghelper.LogSlotError(strconv.FormatUint(ps.Slot, 10), err).Error("Unable to unmarshal the BeaconState.")
		return err
	}

	ps.FullBeaconState = &beaconState
	ps.SszBeaconState = sszBeaconState
	return nil
}

// Check to make sure that the previous block we processed is the parent of the current block.
func (ps *ProcessSlot) checkPreviousSlot(tx sql.Tx, ctx context.Context, previousSlot uint64, previousBlockRoot string, knownGapsTableIncrement int) {
	if nil == ps.FullSignedBeaconBlock {
		log.Debug("Can't check block root, no current block.")
		return
	}
	parentRoot := toHex(ps.FullSignedBeaconBlock.Block().ParentRoot())
	slot := ps.Slot
	if previousSlot == slot {
		log.WithFields(log.Fields{
			"slot": slot,
			"fork": true,
		}).Warn("A fork occurred! The previous slot and current slot match.")
		transactReorgs(tx, ctx, strconv.FormatUint(ps.Slot, 10), ps.BlockRoot, ps.Metrics)
	} else if previousSlot > slot {
		log.WithFields(log.Fields{
			"previousSlot": previousSlot,
			"curSlot":      slot,
		}).Warn("We noticed the previous slot is greater than the current slot.")
	} else if previousSlot+1 != slot {
		log.WithFields(log.Fields{
			"previousSlot": previousSlot,
			"currentSlot":  slot,
		}).Error("We skipped a few slots.")
		transactKnownGaps(tx, ctx, knownGapsTableIncrement, previousSlot+1, slot-1, fmt.Errorf("gaps during head processing"), "headGaps", ps.Metrics)
	} else if previousBlockRoot != parentRoot {
		log.WithFields(log.Fields{
			"previousBlockRoot":  previousBlockRoot,
			"currentBlockParent": parentRoot,
		}).Error("The previousBlockRoot does not match the current blocks parent, an unprocessed fork might have occurred.")
		transactReorgs(tx, ctx, strconv.FormatUint(previousSlot, 10), parentRoot, ps.Metrics)
	} else {
		log.Debug("Previous Slot and Current Slot are one distance from each other.")
	}
}

// Transforms all the raw data into DB models that can be written to the DB.
func (ps *ProcessSlot) createWriteObjects() (*DatabaseWriter, error) {
	var status string
	if ps.Status != "" {
		status = ps.Status
	} else {
		status = "proposed"
	}

	parseBeaconTime := time.Now()
	// These will normally be pre-calculated by this point.
	blockRoot, stateRoot, eth1DataBlockHash, err := ps.provideFinalHash()
	if err != nil {
		return nil, err
	}
	ps.PerformanceMetrics.ParseBeaconObjectForHash = time.Since(parseBeaconTime)

	payloadHeader := ps.provideExecutionPayloadDetails()

	dw, err := CreateDatabaseWrite(ps.Db, ps.Slot, stateRoot, blockRoot, ps.ParentBlockRoot, eth1DataBlockHash,
		payloadHeader, status, &ps.SszSignedBeaconBlock, &ps.SszBeaconState, ps.Metrics)
	if err != nil {
		return dw, err
	}

	return dw, nil
}

// This function will return the final blockRoot, stateRoot, and eth1DataBlockHash that will be
// used to write to a DB
func (ps *ProcessSlot) provideFinalHash() (string, string, string, error) {
	var (
		stateRoot         string
		blockRoot         string
		eth1DataBlockHash string
	)
	if ps.Status == "skipped" {
		stateRoot = ""
		blockRoot = ""
		eth1DataBlockHash = ""
	} else {
		if ps.StateRoot != "" {
			stateRoot = ps.StateRoot
		} else {
			if nil != ps.FullSignedBeaconBlock {
				stateRoot = toHex(ps.FullSignedBeaconBlock.Block().StateRoot())
				log.Debug("BeaconBlock StateRoot: ", stateRoot)
			} else {
				log.Debug("BeaconBlock StateRoot: <nil beacon block>")
			}
		}

		if ps.BlockRoot != "" {
			blockRoot = ps.BlockRoot
		} else {
			if nil != ps.FullSignedBeaconBlock {
				rawBlockRoot := ps.FullSignedBeaconBlock.Block().HashTreeRoot()
				blockRoot = toHex(rawBlockRoot)
				log.WithFields(log.Fields{"blockRoot": blockRoot}).Debug("Block Root from ssz")
			} else {
				log.Debug("BeaconBlock HashTreeRoot: <nil beacon block>")
			}
		}
		if nil != ps.FullSignedBeaconBlock {
			eth1DataBlockHash = toHex(ps.FullSignedBeaconBlock.Block().Body().Eth1Data().BlockHash)
		}
	}
	return blockRoot, stateRoot, eth1DataBlockHash, nil
}

func (ps *ProcessSlot) provideExecutionPayloadDetails() *ExecutionPayloadHeader {
	if nil == ps.FullSignedBeaconBlock || !ps.FullSignedBeaconBlock.IsBellatrix() {
		return nil
	}

	payload := ps.FullSignedBeaconBlock.Block().Body().ExecutionPayloadHeader()
	blockNumber := uint64(payload.BlockNumber)

	// The earliest blocks on the Bellatrix fork, pre-Merge, have zeroed ExecutionPayloads.
	// There is nothing useful to to store in that case, even though the structure exists.
	if blockNumber == 0 {
		return nil
	}

	return payload
}

func toHex(r [32]byte) string {
	return "0x" + hex.EncodeToString(r[:])
}
