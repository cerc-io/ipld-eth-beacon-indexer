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
func (bc *BeaconClient) handleReorg() {
	log.Info("Starting to process reorgs.")
	for {
		reorg := <-bc.ReOrgTracking.ProcessCh
		log.WithFields(log.Fields{"reorg": reorg}).Debug("Received a new reorg message.")
		writeReorgs(bc.Db, reorg.Slot, reorg.NewHeadBlock, bc.Metrics)
	}
}

// This function will handle the latest head event.
func (bc *BeaconClient) handleHead() {
	log.Info("Starting to process head.")
	errorSlots := 0
	for {
		head := <-bc.HeadTracking.ProcessCh
		// Process all the work here.
		slot, err := strconv.Atoi(head.Slot)
		if err != nil {
			bc.HeadTracking.ErrorCh <- &SseError{
				err: fmt.Errorf("Unable to turn the slot from string to int: %s", head.Slot),
			}
			errorSlots = errorSlots + 1
			continue
		}
		if errorSlots != 0 && bc.PreviousSlot != 0 {
			log.WithFields(log.Fields{
				"lastProcessedSlot": bc.PreviousSlot,
				"errorMessages":     errorSlots,
			}).Warn("We added slots to the knownGaps table because we got bad head messages.")
			writeKnownGaps(bc.Db, bc.KnownGapTableIncrement, bc.PreviousSlot, bcSlotsPerEpoch+errorSlots, fmt.Errorf("Bad Head Messages"), "headProcessing")
		}

		err = processHeadSlot(bc.Db, bc.ServerEndpoint, slot, head.Block, head.State, bc.PreviousSlot, bc.PreviousBlockRoot, bc.Metrics, bc.KnownGapTableIncrement)
		if err != nil {
			loghelper.LogSlotError(head.Slot, err).Error("Unable to process a slot")
		}
		log.WithFields(log.Fields{"head": head}).Debug("Received a new head event.")

		// Update the previous block
		bc.PreviousSlot = slot
		bc.PreviousBlockRoot = head.Block
	}

}
