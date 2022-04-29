// This file will keep track of all the code needed to process a slot.
// To process a slot, it should have all the necessary data needed to write it to the DB.
// But not actually write it.

package beaconclient

import (
	"encoding/hex"
	"fmt"

	"github.com/ferranbt/fastssz/spectests"
	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

var (
	SlotUnmarshalError       = "Unable to properly unmarshal the Slot field in the SignedBeaconBlock."
	ParentRootUnmarshalError = "Unable to properly unmarshal the ParentRoot field in the SignedBeaconBlock."
	MissingIdentifiedError   = "Can't Query state without a set slot or block_root"
)

type ProcessSlot struct {
	Slot                  string                       // The slot number.
	BlockRoot             string                       // The hex encoded string of the BlockRoot.
	StateRoot             string                       // The hex encoded string of the StateRoot.
	ParentBlockRoot       string                       // The hex encoded string of the parent block.
	SszSignedBeaconBlock  []byte                       // The entire SSZ encoded SignedBeaconBlock
	SszBeaconState        []byte                       // The entire SSZ encoded BeaconState
	FullBeaconState       *spectests.BeaconState       // The unmarshaled BeaconState object, the unmarshalling could have errors.
	FullSignedBeaconBlock *spectests.SignedBeaconBlock // The unmarshaled BeaconState object, the unmarshalling could have errors.
}

// This function will do all the work to process the slot at head.
func processHeadSlot(baseEndpoint string, slot string, blockRoot string, stateRoot string, parentBlockRoot string, previousSlot uint64, previousBlockRoot string) error {
	pc := &ProcessSlot{
		Slot:            slot,
		BlockRoot:       blockRoot,
		StateRoot:       stateRoot,
		ParentBlockRoot: parentBlockRoot,
	}
	err := pc.getSignedBeaconBlock(baseEndpoint)
	if err != nil {
		return err
	}

	err = pc.getBeaconState(baseEndpoint)
	if err != nil {
		return err
	}

	// Handle any reorgs or skipped slots.
	if previousSlot != 0 && previousBlockRoot != "" {
		pc.checkPreviousSlot(previousSlot, previousBlockRoot)
	}

	// Get this object ready to write

	// Write the object to the DB.

	return nil
}

// Update the SszSignedBeaconBlock and FullSignedBeaconBlock object with their respective values.
func (ps *ProcessSlot) getSignedBeaconBlock(baseEndpoint string) error {
	var blockIdentifier string // Used to query the block
	if ps.BlockRoot != "" {
		blockIdentifier = ps.BlockRoot
	} else if ps.Slot != "" {
		blockIdentifier = ps.Slot
	} else {
		log.Error(MissingIdentifiedError)
		return fmt.Errorf(MissingIdentifiedError)
	}
	blockEndpoint := baseEndpoint + blockIdentifier
	ps.SszSignedBeaconBlock, _ = querySsz(blockEndpoint, ps.Slot)

	ps.FullSignedBeaconBlock = new(spectests.SignedBeaconBlock)
	err := ps.FullSignedBeaconBlock.UnmarshalSSZ(ps.SszSignedBeaconBlock)

	if err != nil {
		if ps.FullSignedBeaconBlock.Block.Slot == 0 {
			loghelper.LogSlotError(ps.Slot, err).Error(SlotUnmarshalError)
			return fmt.Errorf(SlotUnmarshalError)
		} else if ps.FullSignedBeaconBlock.Block.ParentRoot == nil {
			loghelper.LogSlotError(ps.Slot, err).Error(ParentRootUnmarshalError)
			return fmt.Errorf(ParentRootUnmarshalError)
		}
	}
	return nil
}

// Update the SszBeaconState and FullBeaconState object with their respective values.
func (ps *ProcessSlot) getBeaconState(baseEndpoint string) error {
	var stateIdentifier string // Used to query the state
	if ps.StateRoot != "" {
		stateIdentifier = ps.BlockRoot
	} else if ps.Slot != "" {
		stateIdentifier = ps.Slot
	} else {
		log.Error(MissingIdentifiedError)
		return fmt.Errorf(MissingIdentifiedError)
	}
	stateEndpoint := baseEndpoint + stateIdentifier
	ps.SszBeaconState, _ = querySsz(stateEndpoint, ps.Slot)

	ps.FullBeaconState = new(spectests.BeaconState)
	err := ps.FullSignedBeaconBlock.UnmarshalSSZ(ps.SszBeaconState)

	if err != nil {
		if ps.FullBeaconState.Slot == 0 {
			loghelper.LogSlotError(ps.Slot, err).Error(SlotUnmarshalError)
			return fmt.Errorf(SlotUnmarshalError)
		}
	}
	return nil
}

func (ps *ProcessSlot) checkPreviousSlot(previousSlot uint64, previousBlockRoot string) {
	if previousSlot == ps.FullBeaconState.Slot {
		log.WithFields(log.Fields{
			"slot": ps.FullBeaconState.Slot,
			"fork": true,
		}).Warn("A fork occurred! The previous slot and current slot match.")
		// Handle Forks
	} else if previousSlot-1 != ps.FullBeaconState.Slot {
		log.WithFields(log.Fields{
			"previousSlot": previousSlot,
			"currentSlot":  ps.FullBeaconState.Slot,
		}).Error("We skipped a few slots.")
		// Call our batch processing function.
	} else if previousBlockRoot != "0x"+hex.EncodeToString(ps.FullSignedBeaconBlock.Block.ParentRoot) {
		log.WithFields(log.Fields{
			"previousBlockRoot":  previousBlockRoot,
			"currentBlockParent": ps.FullSignedBeaconBlock.Block.ParentRoot,
		}).Error("The previousBlockRoot does not match the current blocks parent, an unprocessed fork might have occurred.")
		// Handle Forks
	}
}
