// This file contains all the functions to handle SSE events after they have been turned
// to the structs.

package beaconclient

import (
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
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

		// Update the previous block if its the first message.
		if bc.PreviousSlot == 0 && bc.PreviousBlockRoot == "" {
			var err error
			bc.PreviousSlot, err = strconv.Atoi(head.Slot)
			if err != nil {
				bc.HeadTracking.ErrorCh <- &SseError{
					err: fmt.Errorf("Unable to turn the slot from string to int: %s", head.Slot),
				}
			}
			bc.PreviousBlockRoot = head.Block
		}
		log.WithFields(log.Fields{"head": head}).Debug("Received a new head event.")
	}

}
