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
	"context"
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/database/sql"
)

// This function will perform the necessary steps to handle a reorg.
func (bc *BeaconClient) handleReorg(ctx context.Context) {
	log.Info("Starting to process reorgs.")
	for {
		select {
		case <-ctx.Done():
			close(bc.ReOrgTracking.ProcessCh)
			return
		case reorg := <-bc.ReOrgTracking.ProcessCh:
			log.WithFields(log.Fields{"reorg": reorg}).Debug("Received a new reorg message.")
			writeReorgs(bc.Db, reorg.Slot, reorg.NewHeadBlock, bc.Metrics)
		}
	}
}

// This function will handle the latest head event.
func (bc *BeaconClient) handleHead(ctx context.Context, maxWorkers int) {
	log.Info("Starting to process head.")

	workCh := make(chan workParams)
	log.WithField("workerNumber", maxWorkers).Info("Creating Workers")
	for i := 1; i < maxWorkers; i++ {
		go bc.headBlockProcessor(ctx, workCh)
	}
	errorSlots := 0
	for {
		select {
		case <-ctx.Done():
			close(bc.HeadTracking.ProcessCh)
			return
		case head := <-bc.HeadTracking.ProcessCh:

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
					"errorSlots":        errorSlots,
				}).Warn("We added slots to the knownGaps table because we got bad head messages.")
				writeKnownGaps(bc.Db, bc.KnownGapTableIncrement, bc.PreviousSlot+1, slot, fmt.Errorf("Bad Head Messages"), "headProcessing", bc.Metrics)
				errorSlots = 0
			}

			log.WithFields(log.Fields{"head": head}).Debug("We are going to start processing the slot.")

			// Not used anywhere yet but might be useful to have.
			if bc.PreviousSlot == 0 && bc.PreviousBlockRoot == "" {
				bc.StartingSlot = slot
			}

			workCh <- workParams{db: bc.Db, serverEndpoint: bc.ServerEndpoint, slot: slot, blockRoot: head.Block, stateRoot: head.State, previousSlot: bc.PreviousSlot, previousBlockRoot: bc.PreviousBlockRoot, metrics: bc.Metrics, knownGapsTableIncrement: bc.KnownGapTableIncrement, checkDb: bc.CheckDb}
			log.WithFields(log.Fields{"head": head.Slot}).Debug("We finished sending this slot to the workCh")

			// Update the previous block
			bc.PreviousSlot = slot
			bc.PreviousBlockRoot = head.Block
		}
	}
}

// A worker that will process head slots.
func (bc *BeaconClient) headBlockProcessor(ctx context.Context, workCh <-chan workParams) {
	for {
		select {
		case <-ctx.Done():
			return
		case wp := <-workCh:
			processHeadSlot(ctx, wp.db, wp.serverEndpoint, wp.slot, wp.blockRoot, wp.stateRoot, wp.previousSlot, wp.previousBlockRoot, wp.metrics, wp.knownGapsTableIncrement, wp.checkDb)
		}
	}
}

// A struct used to pass parameters to the worker.
type workParams struct {
	db                      sql.Database
	serverEndpoint          string
	slot                    int
	blockRoot               string
	stateRoot               string
	previousSlot            int
	previousBlockRoot       string
	metrics                 *BeaconClientMetrics
	knownGapsTableIncrement int
	checkDb                 bool
}
