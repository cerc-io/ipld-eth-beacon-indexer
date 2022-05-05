// This file will keep track of all the code needed to process a slot.
// To process a slot, it should have all the necessary data needed to write it to the DB.
// But not actually write it.

package beaconclient

import (
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
)

var (
	SlotUnmarshalError = func(obj string) string {
		return fmt.Sprintf("Unable to properly unmarshal the Slot field in the %s.", obj)
	}
	ParentRootUnmarshalError = "Unable to properly unmarshal the ParentRoot field in the SignedBeaconBlock."
	MissingIdentifiedError   = "Can't query state without a set slot or block_root"
)

type ProcessSlot struct {
	// Generic

	Slot            int    // The slot number.
	Epoch           int    // The epoch number.
	BlockRoot       string // The hex encoded string of the BlockRoot.
	StateRoot       string // The hex encoded string of the StateRoot.
	ParentBlockRoot string // The hex encoded string of the parent block.
	Status          string // The status of the block
	HeadOrHistoric  string // Is this the head or a historic slot. This is critical when trying to analyze errors and missed slots.
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
func handleFullSlot(db sql.Database, serverAddress string, slot int, blockRoot string, stateRoot string, previousSlot uint64, previousBlockRoot string, headOrHistoric string) error {
	headOrHistoric = strings.ToLower(headOrHistoric)
	if headOrHistoric != "head" && headOrHistoric != "historic" {
		return fmt.Errorf("headOrBatch must be either historic or head!")
	}
	ps := &ProcessSlot{
		Slot:           slot,
		BlockRoot:      blockRoot,
		StateRoot:      stateRoot,
		HeadOrHistoric: headOrHistoric,
	}

	// Get the SignedBeaconBlock.
	err := ps.getSignedBeaconBlock(serverAddress)
	if err != nil {
		return err
	}

	// Get the BeaconState.
	err = ps.getBeaconState(serverAddress)
	if err != nil {
		return err
	}

	// Handle any reorgs or skipped slots.
	if ps.HeadOrHistoric == "head" {
		if previousSlot != 0 && previousBlockRoot != "" {
			ps.checkPreviousSlot(previousSlot, previousBlockRoot)
		}
	}

	// Get this object ready to write
	dw := ps.createWriteObjects(db)

	// Write the object to the DB.
	dw.writeFullSlot()

	return nil
}

// Handle a slot that is at head. A wrapper function for calling `handleFullSlot`.
func handleHeadSlot(db sql.Database, serverAddress string, slot int, blockRoot string, stateRoot string, previousSlot uint64, previousBlockRoot string) error {
	return handleFullSlot(db, serverAddress, slot, blockRoot, stateRoot, previousSlot, previousBlockRoot, "head")
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
	blockEndpoint := serverAddress + bcBlockQueryEndpoint + blockIdentifier
	var err error
	var rc int
	ps.SszSignedBeaconBlock, rc, err = querySsz(blockEndpoint, strconv.Itoa(ps.Slot))
	if err != nil {
		loghelper.LogSlotError(strconv.Itoa(ps.Slot), err).Error("Unable to properly query the slot.")
		return err
	}

	if rc != 200 {
		ps.checkMissedSlot()
	}

	ps.FullSignedBeaconBlock = &st.SignedBeaconBlock{}
	err = ps.FullSignedBeaconBlock.UnmarshalSSZ(ps.SszSignedBeaconBlock)

	if err != nil {
		if ps.FullSignedBeaconBlock.Block.Slot == 0 {
			loghelper.LogSlotError(strconv.Itoa(ps.Slot), err).Error(SlotUnmarshalError("SignedBeaconBlock"))
			return fmt.Errorf(SlotUnmarshalError("SignedBeaconBlock"))
		} else if ps.FullSignedBeaconBlock.Block.ParentRoot == nil {
			loghelper.LogSlotError(strconv.Itoa(ps.Slot), err).Error(ParentRootUnmarshalError)
			return fmt.Errorf(ParentRootUnmarshalError)
		}
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
	stateEndpoint := serverEndpoint + bcStateQueryEndpoint + stateIdentifier
	ps.SszBeaconState, _, _ = querySsz(stateEndpoint, strconv.Itoa(ps.Slot))

	ps.FullBeaconState = new(st.BeaconState)
	err := ps.FullBeaconState.UnmarshalSSZ(ps.SszBeaconState)

	if err != nil {
		if ps.FullBeaconState.Slot == 0 {
			loghelper.LogSlotError(strconv.Itoa(ps.Slot), err).Error(SlotUnmarshalError("BeaconState"))
			return fmt.Errorf(SlotUnmarshalError("BeaconState"))
		}
	}
	return nil
}

// Check to make sure that the previous block we processed is the parent of the current block.
func (ps *ProcessSlot) checkPreviousSlot(previousSlot uint64, previousBlockRoot string) {
	if previousSlot == uint64(ps.FullBeaconState.Slot) {
		log.WithFields(log.Fields{
			"slot": ps.FullBeaconState.Slot,
			"fork": true,
		}).Warn("A fork occurred! The previous slot and current slot match.")
		// mark old slot as forked.
	} else if previousSlot-1 != uint64(ps.FullBeaconState.Slot) {
		log.WithFields(log.Fields{
			"previousSlot": previousSlot,
			"currentSlot":  ps.FullBeaconState.Slot,
		}).Error("We skipped a few slots.")
		// Check to see if the slot was skipped.
		// Call our batch processing function.
	} else if previousBlockRoot != "0x"+hex.EncodeToString(ps.FullSignedBeaconBlock.Block.ParentRoot) {
		log.WithFields(log.Fields{
			"previousBlockRoot":  previousBlockRoot,
			"currentBlockParent": ps.FullSignedBeaconBlock.Block.ParentRoot,
		}).Error("The previousBlockRoot does not match the current blocks parent, an unprocessed fork might have occurred.")
		// Handle Forks
		// Mark the previous slot in the DB as a fork.
		// Continue with this slot.
	} else {
		log.Debug("Previous Slot and Current Slot are one distance from each other.")
	}
}

// Add logic for checking a missed Slot
// IF the state is present but block is not, then it was skipped???
// If the state and block are both absent, then the block might be missing??
// IF state is absent but block is not, there might be an issue with the LH client.
// Check the previous and following slot?
// Check if head or historic.

// 1. BeaconBlock is 404.
// 2. check heck /lighthouse/database/info to make sure the oldest_block_slot == 0  and anchor == null. This indicates that I don't have any gaps in the DB.
// 3. I query BeaconState for slot X, and get a BeaconState.
// 4. Although for good measure you should also check that the head is at a slot >= X using something like /eth/v1/node/syncing/ or /eth/v1/beacon/headers/head
func (ps *ProcessSlot) checkMissedSlot() {

}

// Transforms all the raw data into DB models that can be written to the DB.
func (ps *ProcessSlot) createWriteObjects(db sql.Database) *DatabaseWriter {
	var (
		stateRoot string
		blockRoot string
		status    string
	)

	if ps.StateRoot != "" {
		stateRoot = ps.StateRoot
	} else {
		stateRoot = "0x" + hex.EncodeToString(ps.FullSignedBeaconBlock.Block.StateRoot)
		log.Debug("StateRoot: ", stateRoot)
	}

	// MUST RESOLVE!
	if ps.BlockRoot != "" {
		blockRoot = ps.BlockRoot
	} else {
		// We need to get the state of Slot + 1, then we can run the below.
		// WE can query it for each run, or we can leave it blank, and update it.
		// I just want to avoid getting the same state twice, especially since the state can get heavy.
		// blockRoot = "0x" + hex.EncodeToString(ps.FullBeaconState.GetBlockRoots()[ps.Slot%bcSlotPerHistoricalVector][:])
		// log.Debug("Block Root: ", blockRoot)
		// log.Debug("ps.Slott: ", ps.Slot)
		// Maybe we can use the helper down the road.
		//blockRootRaw, _ := helper.BlockRootAtSlot(ps.FullBeaconState, ps.FullSignedBeaconBlock.Block.Slot)
		//blockRoot = string(blockRootRaw)
	}

	if ps.Status != "" {
		status = ps.Status
	} else {
		status = "proposed"
	}

	dw := CreateDatabaseWrite(db, ps.Slot, stateRoot, blockRoot, ps.ParentBlockRoot, status)
	dw.rawSignedBeaconBlock = ps.SszSignedBeaconBlock
	dw.rawBeaconState = ps.SszBeaconState

	return dw
}
