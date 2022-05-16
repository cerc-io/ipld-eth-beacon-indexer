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

	// The below is temporary, once https://github.com/prysmaticlabs/prysm/issues/10006 has been resolved we wont need it.
	// pb "github.com/prysmaticlabs/prysm/proto/prysm/v2"

	st "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
	"golang.org/x/sync/errgroup"
)

var (
	SlotUnmarshalError = func(obj string) string {
		return fmt.Sprintf("Unable to properly unmarshal the Slot field in the %s.", obj)
	}
	ParentRootUnmarshalError = "Unable to properly unmarshal the ParentRoot field in the SignedBeaconBlock."
	MissingIdentifiedError   = "Can't query state without a set slot or block_root"
	MissingEth1Data          = "Can't get the Eth1 block_hash"
)

type ProcessSlot struct {
	// Generic

	Slot            int                  // The slot number.
	Epoch           int                  // The epoch number.
	BlockRoot       string               // The hex encoded string of the BlockRoot.
	StateRoot       string               // The hex encoded string of the StateRoot.
	ParentBlockRoot string               // The hex encoded string of the parent block.
	Status          string               // The status of the block
	HeadOrHistoric  string               // Is this the head or a historic slot. This is critical when trying to analyze errors and skipped slots.
	Db              sql.Database         // The DB object used to write to the DB.
	Metrics         *BeaconClientMetrics // An object to keep track of the beaconclient metrics
	// BeaconBlock

	SszSignedBeaconBlock  []byte                // The entire SSZ encoded SignedBeaconBlock
	FullSignedBeaconBlock *st.SignedBeaconBlock // The unmarshaled BeaconState object, the unmarshalling could have errors.

	// BeaconState
	FullBeaconState *st.BeaconState // The unmarshaled BeaconState object, the unmarshalling could have errors.
	SszBeaconState  []byte          // The entire SSZ encoded BeaconState

	// DB Write objects
	DbSlotsModel             *DbSlots             // The model being written to the slots table.
	DbSignedBeaconBlockModel *DbSignedBeaconBlock // The model being written to the signed_beacon_block table.
	DbBeaconState            *DbBeaconState       // The model being written to the beacon_state table.
}

// This function will do all the work to process the slot and write it to the DB.
func processFullSlot(db sql.Database, serverAddress string, slot int, blockRoot string, stateRoot string, previousSlot int, previousBlockRoot string, headOrHistoric string, metrics *BeaconClientMetrics, knownGapsTableIncrement int) error {
	ps := &ProcessSlot{
		Slot:           slot,
		BlockRoot:      blockRoot,
		StateRoot:      stateRoot,
		HeadOrHistoric: headOrHistoric,
		Db:             db,
		Metrics:        metrics,
	}

	g, _ := errgroup.WithContext(context.Background())

	// Get the BeaconState.
	g.Go(func() error {
		err := ps.getBeaconState(serverAddress)
		if err != nil {
			return err
		}
		return nil
	})

	// Get the SignedBeaconBlock.
	g.Go(func() error {
		err := ps.getSignedBeaconBlock(serverAddress)
		if err != nil {
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		writeKnownGaps(ps.Db, 1, ps.Slot, ps.Slot, err, "processSlot")
	}

	if ps.HeadOrHistoric == "head" && previousSlot == 0 && previousBlockRoot == "" {
		writeStartUpGaps(db, knownGapsTableIncrement, ps.Slot)
	}

	// Get this object ready to write
	blockRootEndpoint := serverAddress + BcBlockRootEndpoint(strconv.Itoa(ps.Slot))
	dw, err := ps.createWriteObjects(blockRootEndpoint)
	if err != nil {
		writeKnownGaps(ps.Db, 1, ps.Slot, ps.Slot, err, "blockRoot")
		return err
	}
	// Write the object to the DB.
	err = dw.writeFullSlot()
	if err != nil {
		writeKnownGaps(ps.Db, 1, ps.Slot, ps.Slot, err, "processSlot")
	}

	// Handle any reorgs or skipped slots.
	headOrHistoric = strings.ToLower(headOrHistoric)
	if headOrHistoric != "head" && headOrHistoric != "historic" {
		return fmt.Errorf("headOrHistoric must be either historic or head!")
	}
	if ps.HeadOrHistoric == "head" && previousSlot != 0 && previousBlockRoot != "" && ps.Status != "skipped" {
		ps.checkPreviousSlot(previousSlot, previousBlockRoot, knownGapsTableIncrement)
	}
	return nil
}

// Handle a slot that is at head. A wrapper function for calling `handleFullSlot`.
func processHeadSlot(db sql.Database, serverAddress string, slot int, blockRoot string, stateRoot string, previousSlot int, previousBlockRoot string, metrics *BeaconClientMetrics, knownGapsTableIncrement int) error {
	return processFullSlot(db, serverAddress, slot, blockRoot, stateRoot, previousSlot, previousBlockRoot, "head", metrics, knownGapsTableIncrement)
}

// Handle a historic slot. A wrapper function for calling `handleFullSlot`.
// Commented because of the linter...... LOL
//func handleHistoricSlot(db sql.Database, serverAddress string, slot int) error {
//	return handleFullSlot(db, serverAddress, slot, "", "", 0, "", "historic")
//}

// Update the SszSignedBeaconBlock and FullSignedBeaconBlock object with their respective values.
func (ps *ProcessSlot) getSignedBeaconBlock(serverAddress string) error {
	var blockIdentifier string // Used to query the block
	if ps.BlockRoot != "" {
		blockIdentifier = ps.BlockRoot
	} else if ps.Slot != 0 {
		blockIdentifier = strconv.Itoa(ps.Slot)
	} else {
		log.Error(MissingIdentifiedError)
		return fmt.Errorf(MissingIdentifiedError)
	}
	blockEndpoint := serverAddress + BcBlockQueryEndpoint + blockIdentifier
	var err error
	var rc int
	ps.SszSignedBeaconBlock, rc, err = querySsz(blockEndpoint, strconv.Itoa(ps.Slot))
	if err != nil {
		loghelper.LogSlotError(strconv.Itoa(ps.Slot), err).Error("Unable to properly query the slot.")
		return err
	}

	if rc != 200 {
		ps.FullSignedBeaconBlock = &st.SignedBeaconBlock{}
		ps.SszSignedBeaconBlock = []byte{}
		ps.ParentBlockRoot = ""
		ps.Status = "skipped"
		return nil
	}

	ps.FullSignedBeaconBlock = &st.SignedBeaconBlock{}
	err = ps.FullSignedBeaconBlock.UnmarshalSSZ(ps.SszSignedBeaconBlock)

	if err != nil {
		loghelper.LogError(err).Debug("We are getting an error message when unmarshalling the SignedBeaconBlock.")
		if ps.FullSignedBeaconBlock.Block.Slot == 0 {
			loghelper.LogSlotError(strconv.Itoa(ps.Slot), err).Error(SlotUnmarshalError("SignedBeaconBlock"))
			return fmt.Errorf(SlotUnmarshalError("SignedBeaconBlock"))
		} else if ps.FullSignedBeaconBlock.Block.ParentRoot == nil {
			loghelper.LogSlotError(strconv.Itoa(ps.Slot), err).Error(ParentRootUnmarshalError)
			return fmt.Errorf(ParentRootUnmarshalError)
		} else if hex.EncodeToString(ps.FullBeaconState.Eth1Data.BlockHash) == "" {
			loghelper.LogSlotError(strconv.Itoa(ps.Slot), err).Error(MissingEth1Data)
			return fmt.Errorf(MissingEth1Data)
		}
		log.Warn("We received a processing error: ", err)
	}
	ps.ParentBlockRoot = "0x" + hex.EncodeToString(ps.FullSignedBeaconBlock.Block.ParentRoot)
	return nil
}

// Update the SszBeaconState and FullBeaconState object with their respective values.
func (ps *ProcessSlot) getBeaconState(serverEndpoint string) error {
	var stateIdentifier string // Used to query the state
	if ps.StateRoot != "" {
		stateIdentifier = ps.StateRoot
	} else if ps.Slot != 0 {
		stateIdentifier = strconv.Itoa(ps.Slot)
	} else {
		log.Error(MissingIdentifiedError)
		return fmt.Errorf(MissingIdentifiedError)
	}
	stateEndpoint := serverEndpoint + BcStateQueryEndpoint + stateIdentifier
	ps.SszBeaconState, _, _ = querySsz(stateEndpoint, strconv.Itoa(ps.Slot))

	ps.FullBeaconState = new(st.BeaconState)
	err := ps.FullBeaconState.UnmarshalSSZ(ps.SszBeaconState)

	if err != nil {
		loghelper.LogError(err).Debug("We are getting an error message when unmarshalling the BeaconState")
		if ps.FullBeaconState.Slot == 0 {
			loghelper.LogSlotError(strconv.Itoa(ps.Slot), err).Error(SlotUnmarshalError("BeaconState"))
			return fmt.Errorf(SlotUnmarshalError("BeaconState"))
		}
	}
	return nil
}

// Check to make sure that the previous block we processed is the parent of the current block.
func (ps *ProcessSlot) checkPreviousSlot(previousSlot int, previousBlockRoot string, knownGapsTableIncrement int) {
	parentRoot := "0x" + hex.EncodeToString(ps.FullSignedBeaconBlock.Block.ParentRoot)
	if previousSlot == int(ps.FullBeaconState.Slot) {
		log.WithFields(log.Fields{
			"slot": ps.FullBeaconState.Slot,
			"fork": true,
		}).Warn("A fork occurred! The previous slot and current slot match.")
		writeReorgs(ps.Db, strconv.Itoa(ps.Slot), ps.BlockRoot, ps.Metrics)
	} else if previousSlot+1 != int(ps.FullBeaconState.Slot) {
		log.WithFields(log.Fields{
			"previousSlot": previousSlot,
			"currentSlot":  ps.FullBeaconState.Slot,
		}).Error("We skipped a few slots.")
		writeKnownGaps(ps.Db, knownGapsTableIncrement, previousSlot+1, int(ps.FullBeaconState.Slot)-1, fmt.Errorf("Gaps during head processing"), "headGaps")
	} else if previousBlockRoot != parentRoot {
		log.WithFields(log.Fields{
			"previousBlockRoot":  previousBlockRoot,
			"currentBlockParent": parentRoot,
		}).Error("The previousBlockRoot does not match the current blocks parent, an unprocessed fork might have occurred.")
		writeReorgs(ps.Db, strconv.Itoa(previousSlot), parentRoot, ps.Metrics)
		writeKnownGaps(ps.Db, 1, ps.Slot-1, ps.Slot-1, fmt.Errorf("Incorrect Parent"), "processSlot")
	} else {
		log.Debug("Previous Slot and Current Slot are one distance from each other.")
	}
}

// Transforms all the raw data into DB models that can be written to the DB.
func (ps *ProcessSlot) createWriteObjects(blockRootEndpoint string) (*DatabaseWriter, error) {
	var (
		stateRoot     string
		blockRoot     string
		status        string
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
			stateRoot = "0x" + hex.EncodeToString(ps.FullSignedBeaconBlock.Block.StateRoot)
			log.Debug("StateRoot: ", stateRoot)
		}

		if ps.BlockRoot != "" {
			blockRoot = ps.BlockRoot
		} else {
			var err error
			blockRoot, err = queryBlockRoot(blockRootEndpoint, strconv.Itoa(ps.Slot))
			if err != nil {
				return nil, err
			}
		}
		eth1BlockHash = "0x" + hex.EncodeToString(ps.FullSignedBeaconBlock.Block.Body.Eth1Data.BlockHash)
	}

	if ps.Status != "" {
		status = ps.Status
	} else {
		status = "proposed"
	}

	dw, err := CreateDatabaseWrite(ps.Db, ps.Slot, stateRoot, blockRoot, ps.ParentBlockRoot, eth1BlockHash, status, ps.SszSignedBeaconBlock, ps.SszBeaconState, ps.Metrics)
	if err != nil {
		return dw, err
	}

	return dw, nil
}
