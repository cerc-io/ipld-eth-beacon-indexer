// This file contains all the functions to handle SSE events after they have been turned
// to the structs.

package beaconclient

import (
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

// This function will perform the necessary steps to handle a reorg.
func (bc *BeaconClient) handleReorgs() {
	log.Info("Starting to process reorgs.")
	for {
		// We will add real functionality later
		reorg := <-bc.ReOrgTracking.ProcessCh
		log.WithFields(log.Fields{"reorg": reorg}).Debug("Received a new reorg message.")
	}
}

// This function will perform the necessary steps to handle a reorg.
func (bc *BeaconClient) handleFinalizedCheckpoint() {
	log.Info("Starting to process finalized checkpoints.")
	for {
		// We will add real functionality later
		finalized := <-bc.ReOrgTracking.ProcessCh
		log.WithFields(log.Fields{"finalized": finalized}).Debug("Received a new finalized checkpoint.")
	}

}

// This function will handle the latest head event.
func (bc *BeaconClient) handleHead() {
	log.Info("Starting to process head.")
	for {
		head := <-bc.HeadTracking.ProcessCh
		// Process all the work here.
		slot, err := strconv.Atoi(head.Slot)
		if err != nil {
			bc.HeadTracking.ErrorCh <- &SseError{
				err: fmt.Errorf("Unable to turn the slot from string to int: %s", head.Slot),
			}
		}
		err = handleHeadSlot(bc.ServerEndpoint, slot, head.Block, head.State, uint64(bc.PreviousSlot), bc.PreviousBlockRoot)
		if err != nil {
			loghelper.LogSlotError(head.Slot, err)
		}
		log.WithFields(log.Fields{"head": head}).Debug("Received a new head event.")

		// Update the previous block
		bc.PreviousSlot = slot
		bc.PreviousBlockRoot = head.Block
	}

}
