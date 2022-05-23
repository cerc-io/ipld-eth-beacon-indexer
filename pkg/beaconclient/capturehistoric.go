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
// This file will call all the functions to start and stop capturing the head of the beacon chain.

package beaconclient

import (
	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"golang.org/x/sync/errgroup"
)

// This function will perform all the heavy lifting for tracking the head of the chain.
func (bc *BeaconClient) CaptureHistoric(maxWorkers int) []error {
	log.Info("We are starting the historical processing service.")
	hp := historicProcessing{db: bc.Db, metrics: bc.Metrics}
	errs := handleBatchProcess(maxWorkers, hp, hp.db, bc.ServerEndpoint, bc.Metrics)
	log.Debug("Exiting Historical")
	return errs
}

// An interface to enforce any batch processing. Currently there are two use cases for this.
//
// 1. Historic Processing
//
// 2. Known Gaps Processing
type BatchProcessing interface {
	getSlotRange(chan<- slotsToProcess) []error // Write the slots to process in a channel, return an error if you cant get the next slots to write.
	handleProcessingErrors(<-chan batchHistoricError)
	removeTableEntry(<-chan slotsToProcess) error // With the provided start and end slot, remove the entry from the database.
}

// A struct to pass around indicating a table entry for slots to process.
type slotsToProcess struct {
	startSlot int // The start slot
	endSlot   int // The end slot
}

type batchHistoricError struct {
	err        error  // The error that occurred when attempting to a slot
	errProcess string // The process that caused the error.
	slot       int    // The slot which the error is for.
}

// Wrapper function for the BatchProcessing interface.
// This function will take the structure that needs batch processing.
// It follows a generic format.
// Get new entries from any given table.
// 1. Add it to the slotsCh.
//
// 2. Run the maximum specified workers to handle individual slots. We need a maximum because we don't want
// To store too many SSZ objects in memory.
//
// 3. Process the slots and send the err to the ErrCh. Each structure can define how it wants its own errors handled.
//
// 4. Remove the slot entry from the DB.
//
// 5. Handle any errors.
func handleBatchProcess(maxWorkers int, bp BatchProcessing, db sql.Database, serverEndpoint string, metrics *BeaconClientMetrics) []error {
	slotsCh := make(chan slotsToProcess)
	workCh := make(chan int)
	processedCh := make(chan slotsToProcess)
	errCh := make(chan batchHistoricError)
	finishCh := make(chan []error, 1)

	// Start workers
	for w := 1; w <= maxWorkers; w++ {
		log.WithFields(log.Fields{"maxWorkers": maxWorkers}).Debug("Starting historic processing workers")
		go processSlotRangeWorker(workCh, errCh, db, serverEndpoint, metrics)
	}

	// Process all ranges and send each individual slot to the worker.
	go func() {
		for slots := range slotsCh {
			for i := slots.startSlot; i <= slots.endSlot; i++ {
				workCh <- i
			}
			processedCh <- slots
		}
	}()

	// Remove entries, end the application if a row cannot be removed..
	go func() {
		errG := new(errgroup.Group)
		errG.Go(func() error {
			return bp.removeTableEntry(processedCh)
		})
		if err := errG.Wait(); err != nil {
			finishCh <- []error{err}
		}
	}()
	// Process errors from slot processing.
	go bp.handleProcessingErrors(errCh)

	// Get slots from the DB.
	go func() {
		errs := bp.getSlotRange(slotsCh) // Periodically adds new entries....
		if errs != nil {
			finishCh <- errs
		}
		finishCh <- nil
	}()

	errs := <-finishCh
	log.Debug("Finishing the batchProcess")
	return errs
}
