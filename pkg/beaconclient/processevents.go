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
// This file contains all the functions to handle SSE events after they have been turned
// to the structs.

package beaconclient

import (
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
	"golang.org/x/sync/errgroup"
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
			writeKnownGaps(bc.Db, bc.KnownGapTableIncrement, bc.PreviousSlot, bcSlotsPerEpoch+errorSlots, fmt.Errorf("Bad Head Messages"), "headProcessing", bc.Metrics)
		}

		log.WithFields(log.Fields{"head": head}).Debug("We are going to start processing the slot.")

		go func(db sql.Database, serverAddress string, slot int, blockRoot string, stateRoot string, previousSlot int, previousBlockRoot string, metrics *BeaconClientMetrics, knownGapsTableIncrement int) {
			errG := new(errgroup.Group)
			errG.Go(func() error {
				err = processHeadSlot(db, serverAddress, slot, blockRoot, stateRoot, previousSlot, previousBlockRoot, metrics, knownGapsTableIncrement)
				if err != nil {
					return err
				}
				return nil
			})
			if err := errG.Wait(); err != nil {
				loghelper.LogSlotError(strconv.Itoa(slot), err).Error("Unable to process a slot")
			}
		}(bc.Db, bc.ServerEndpoint, slot, head.Block, head.State, bc.PreviousSlot, bc.PreviousBlockRoot, bc.Metrics, bc.KnownGapTableIncrement)

		log.WithFields(log.Fields{"head": head.Slot}).Debug("We finished calling processHeadSlot.")

		// Update the previous block
		bc.PreviousSlot = slot
		bc.PreviousBlockRoot = head.Block
	}

}
