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
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/loghelper"
	"golang.org/x/sync/errgroup"
)

type ProcessSlot struct {
	// Generic

	Slot               int                  // The slot number.
	Epoch              int                  // The epoch number.
	BlockRoot          string               // The hex encoded string of the BlockRoot.
	StateRoot          string               // The hex encoded string of the StateRoot.
	ParentBlockRoot    string               // The hex encoded string of the parent block.
	Status             string               // The status of the block
	HeadOrHistoric     string               // Is this the head or a historic slot. This is critical when trying to analyze errors and skipped slots.
	Db                 sql.Database         // The DB object used to write to the DB.
	Metrics            *BeaconClientMetrics // An object to keep track of the beaconclient metrics
	PerformanceMetrics PerformanceMetrics   // An object to keep track of performance metrics.
	// BeaconBlock

	SszSignedBeaconBlock  []byte            // The entire SSZ encoded SignedBeaconBlock
	FullSignedBeaconBlock SignedBeaconBlock // The unmarshaled BeaconState object, the unmarshalling could have errors.

	// BeaconState
	FullBeaconState BeaconState // The unmarshaled BeaconState object, the unmarshalling could have errors.
	SszBeaconState  []byte      // The entire SSZ encoded BeaconState

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
func processFullSlot(ctx context.Context, db sql.Database, serverAddress string, slot int, blockRoot string, stateRoot string, previousSlot int, previousBlockRoot string, headOrHistoric string, metrics *BeaconClientMetrics, knownGapsTableIncrement int, checkDb bool) (error, string) {
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
			Db:             db,
			Metrics:        metrics,
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

		// Get the BeaconState.
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return nil
			default:
				start := time.Now()
				err := ps.getBeaconState(serverAddress)
				if err != nil {
					return err
				}
				ps.PerformanceMetrics.BeaconNodeStateRetrievalTime = time.Since(start)
				return nil
			}
		})

		// Get the SignedBeaconBlock.
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return nil
			default:
				start := time.Now()
				err := ps.getSignedBeaconBlock(serverAddress)
				if err != nil {
					return err
				}
				ps.PerformanceMetrics.BeaconNodeBlockRetrievalTime = time.Since(start)
				return nil
			}
		})

		if err := g.Wait(); err != nil {
			return err, "processSlot"
		}

		parseBeaconTime := time.Now()
		finalBlockRoot, finalStateRoot, finalEth1BlockHash, err := ps.provideFinalHash()
		if err != nil {
			return err, "CalculateBlockRoot"
		}
		ps.PerformanceMetrics.ParseBeaconObjectForHash = time.Since(parseBeaconTime)

		if checkDb {
			checkDbTime := time.Now()
			inDb, err := IsSlotInDb(ctx, ps.Db, strconv.Itoa(ps.Slot), finalBlockRoot, finalStateRoot)
			if err != nil {
				return err, "checkDb"
			}
			if inDb {
				log.WithField("slot", slot).Info("Slot already in the DB.")
				return nil, ""
			}
			ps.PerformanceMetrics.CheckDbPreProcessing = time.Since(checkDbTime)
		}

		// Get this object ready to write
		createDbWriteTime := time.Now()
		dw, err := ps.createWriteObjects(finalBlockRoot, finalStateRoot, finalEth1BlockHash)
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
func processHeadSlot(db sql.Database, serverAddress string, slot int, blockRoot string, stateRoot string, previousSlot int, previousBlockRoot string, metrics *BeaconClientMetrics, knownGapsTableIncrement int, checkDb bool) {
	// Get the knownGaps at startUp.
	if previousSlot == 0 && previousBlockRoot == "" {
		writeStartUpGaps(db, knownGapsTableIncrement, slot, metrics)
	}
	err, errReason := processFullSlot(context.Background(), db, serverAddress, slot, blockRoot, stateRoot, previousSlot, previousBlockRoot, "head", metrics, knownGapsTableIncrement, checkDb)
	if err != nil {
		writeKnownGaps(db, knownGapsTableIncrement, slot, slot, err, errReason, metrics)
	}
}

// Handle a historic slot. A wrapper function for calling `handleFullSlot`.
func handleHistoricSlot(ctx context.Context, db sql.Database, serverAddress string, slot int, metrics *BeaconClientMetrics, checkDb bool) (error, string) {
	return processFullSlot(ctx, db, serverAddress, slot, "", "", 0, "", "historic", metrics, 1, checkDb)
}

// Update the SszSignedBeaconBlock and FullSignedBeaconBlock object with their respective values.
func (ps *ProcessSlot) getSignedBeaconBlock(serverAddress string) error {
	var blockIdentifier string // Used to query the block
	if ps.BlockRoot != "" {
		blockIdentifier = ps.BlockRoot
	} else {
		blockIdentifier = strconv.Itoa(ps.Slot)
	}

	blockEndpoint := serverAddress + BcBlockQueryEndpoint + blockIdentifier
	sszSignedBeaconBlock, rc, err := querySsz(blockEndpoint, strconv.Itoa(ps.Slot))

	if err != nil || rc != 200 {
		loghelper.LogSlotError(strconv.Itoa(ps.Slot), err).Error("Unable to properly query the slot.")
		ps.FullSignedBeaconBlock = SignedBeaconBlock{}
		ps.SszSignedBeaconBlock = []byte{}
		ps.ParentBlockRoot = ""
		ps.Status = "skipped"
		return nil
	}

	var signedBeaconBlock SignedBeaconBlock
	err = signedBeaconBlock.UnmarshalSSZ(sszSignedBeaconBlock)
	if err != nil {
		loghelper.LogSlotError(strconv.Itoa(ps.Slot), err).Error("Unable to unmarshal SignedBeaconBlock for slot.")
		ps.FullSignedBeaconBlock = SignedBeaconBlock{}
		ps.SszSignedBeaconBlock = []byte{}
		ps.ParentBlockRoot = ""
		ps.Status = "skipped"
		return nil
	}

	ps.FullSignedBeaconBlock = signedBeaconBlock
	ps.SszSignedBeaconBlock = sszSignedBeaconBlock

	ps.ParentBlockRoot = rootToHex(ps.FullSignedBeaconBlock.Block().ParentRoot())
	return nil
}

// Update the SszBeaconState and FullBeaconState object with their respective values.
func (ps *ProcessSlot) getBeaconState(serverEndpoint string) error {
	var stateIdentifier string // Used to query the state
	if ps.StateRoot != "" {
		stateIdentifier = ps.StateRoot
	} else {
		stateIdentifier = strconv.Itoa(ps.Slot)
	}

	stateEndpoint := serverEndpoint + BcStateQueryEndpoint + stateIdentifier
	sszBeaconState, _, err := querySsz(stateEndpoint, strconv.Itoa(ps.Slot))
	if err != nil {
		loghelper.LogSlotError(strconv.Itoa(ps.Slot), err).Error("Unable to properly query the BeaconState.")
		return err
	}

	var beaconState BeaconState
	err = beaconState.UnmarshalSSZ(sszBeaconState)
	if err != nil {
		loghelper.LogSlotError(strconv.Itoa(ps.Slot), err).Error("Unable to unmarshal the BeaconState.")
		return err
	}

	ps.FullBeaconState = beaconState
	ps.SszBeaconState = sszBeaconState
	return nil
}

// Check to make sure that the previous block we processed is the parent of the current block.
func (ps *ProcessSlot) checkPreviousSlot(tx sql.Tx, ctx context.Context, previousSlot int, previousBlockRoot string, knownGapsTableIncrement int) {
	parentRoot := rootToHex(ps.FullSignedBeaconBlock.Block().ParentRoot())
	slot := int(ps.FullBeaconState.Slot())
	if previousSlot == slot {
		log.WithFields(log.Fields{
			"slot": slot,
			"fork": true,
		}).Warn("A fork occurred! The previous slot and current slot match.")
		transactReorgs(tx, ctx, strconv.Itoa(ps.Slot), ps.BlockRoot, ps.Metrics)
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
		transactReorgs(tx, ctx, strconv.Itoa(previousSlot), parentRoot, ps.Metrics)
	} else {
		log.Debug("Previous Slot and Current Slot are one distance from each other.")
	}
}

// Transforms all the raw data into DB models that can be written to the DB.
func (ps *ProcessSlot) createWriteObjects(blockRoot, stateRoot, eth1BlockHash string) (*DatabaseWriter, error) {
	var status string
	if ps.Status != "" {
		status = ps.Status
	} else {
		status = "proposed"
	}

	dw, err := CreateDatabaseWrite(ps.Db, ps.Slot, stateRoot, blockRoot, ps.ParentBlockRoot, eth1BlockHash, status, &ps.SszSignedBeaconBlock, &ps.SszBeaconState, ps.Metrics)
	if err != nil {
		return dw, err
	}

	return dw, nil
}

// This function will return the final blockRoot, stateRoot, and eth1BlockHash that will be
// used to write to a DB
func (ps *ProcessSlot) provideFinalHash() (string, string, string, error) {
	var (
		stateRoot     string
		blockRoot     string
		eth1BlockHash string
	)
	if ps.Status == "skipped" {
		stateRoot = ""
		blockRoot = ""
		eth1BlockHash = ""
	} else {
		if ps.StateRoot != "" {
			stateRoot = ps.StateRoot
		} else {
			stateRoot = rootToHex(ps.FullSignedBeaconBlock.Block().StateRoot())
			log.Debug("StateRoot: ", stateRoot)
		}

		if ps.BlockRoot != "" {
			blockRoot = ps.BlockRoot
		} else {
			rawBlockRoot := ps.FullSignedBeaconBlock.Block().HashTreeRoot()
			blockRoot = rootToHex(&rawBlockRoot)
			log.WithFields(log.Fields{"blockRoot": blockRoot}).Debug("Block Root from ssz")
		}
		eth1BlockHash = rootToHex(&ps.FullSignedBeaconBlock.Block().Body().Eth1Data().BlockHash)
	}
	return blockRoot, stateRoot, eth1BlockHash, nil
}

func rootToHex(r *common.Root) string {
	return "0x" + hex.EncodeToString(r[:])
}
