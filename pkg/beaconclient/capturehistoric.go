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
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/loghelper"
	"golang.org/x/sync/errgroup"
)

// This function will perform all the heavy lifting for tracking the head of the chain.
func (bc *BeaconClient) CaptureHistoric(ctx context.Context, maxWorkers int, minimumSlot uint64) []error {
	log.Info("We are starting the historical processing service.")
	bc.HistoricalProcess = HistoricProcessing{db: bc.Db, metrics: bc.Metrics, uniqueNodeIdentifier: bc.UniqueNodeIdentifier}
	errs := handleBatchProcess(ctx, maxWorkers, bc.HistoricalProcess, bc.SlotProcessingDetails(), bc.Metrics.IncrementHistoricSlotProcessed, minimumSlot)
	log.Debug("Exiting Historical")
	return errs
}

// This function will perform all the necessary clean up tasks for stopping historical processing.
func (bc *BeaconClient) StopHistoric(cancel context.CancelFunc) error {
	log.Info("We are stopping the historical processing service.")
	cancel()
	err := bc.HistoricalProcess.releaseDbLocks()
	if err != nil {
		loghelper.LogError(err).WithField("uniqueIdentifier", bc.UniqueNodeIdentifier).Error("We were unable to remove the locks from the eth_beacon.historic_processing table. Manual Intervention is needed!")
	}
	return nil
}

// An interface to enforce any batch processing. Currently there are two use cases for this.
//
// 1. Historic Processing
//
// 2. Known Gaps Processing
type BatchProcessing interface {
	getSlotRange(context.Context, chan<- slotsToProcess, uint64) []error // Write the slots to process in a channel, return an error if you cant get the next slots to write.
	handleProcessingErrors(context.Context, <-chan batchHistoricError)   // Custom logic to handle errors.
	removeTableEntry(context.Context, <-chan slotsToProcess) error       // With the provided start and end slot, remove the entry from the database.
	releaseDbLocks() error                                               // Update the checked_out column to false for whatever table is being updated.
}

/// ^^^
// Might be better to remove the interface and create a single struct that historicalProcessing
// and knownGapsProcessing can use. The struct would contain all the SQL strings that they need.
// And the only difference in logic for processing would be within the error handling.
// Which can be a function we pass into handleBatchProcess()

// A struct to pass around indicating a table entry for slots to process.
type slotsToProcess struct {
	startSlot uint64 // The start slot
	endSlot   uint64 // The end slot
}

type batchHistoricError struct {
	err        error  // The error that occurred when attempting to a slot
	errProcess string // The process that caused the error.
	slot       uint64 // The slot which the error is for.
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
func handleBatchProcess(ctx context.Context, maxWorkers int, bp BatchProcessing, spd SlotProcessingDetails, incrementTracker func(uint64), minimumSlot uint64) []error {
	slotsCh := make(chan slotsToProcess)
	workCh := make(chan uint64)
	processedCh := make(chan slotsToProcess)
	errCh := make(chan batchHistoricError)
	finalErrCh := make(chan []error, 1)

	// Checkout Rows with same node Identifier.
	err := bp.releaseDbLocks()
	if err != nil {
		loghelper.LogError(err).Error(("We are unable to un-checkout entries at the start!"))
	}

	// Start workers
	for w := 1; w <= maxWorkers; w++ {
		log.WithFields(log.Fields{"maxWorkers": maxWorkers}).Debug("Starting batch processing workers")

		go processSlotRangeWorker(ctx, workCh, errCh, spd, incrementTracker)
	}

	// Process all ranges and send each individual slot to the worker.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case slots := <-slotsCh:
				if slots.startSlot > slots.endSlot {
					log.Error("We received a batch process request where the startSlot is greater than the end slot.")
					errCh <- batchHistoricError{
						err:        fmt.Errorf("We received a startSlot where the start was greater than the end."),
						errProcess: "RangeOrder",
						slot:       slots.startSlot,
					}
					errCh <- batchHistoricError{
						err:        fmt.Errorf("We received a endSlot where the start was greater than the end."),
						errProcess: "RangeOrder",
						slot:       slots.endSlot,
					}
				} else if slots.startSlot == slots.endSlot {
					log.WithField("slot", slots.startSlot).Debug("Added new slot to workCh")
					workCh <- slots.startSlot
					processedCh <- slots
				} else {
					for i := slots.startSlot; i <= slots.endSlot; i++ {
						workCh <- i
						log.WithField("slot", i).Debug("Added new slot to workCh")
					}
					processedCh <- slots
				}
			}

		}
	}()

	// Remove entries, end the application if a row cannot be removed..
	go func() {
		errG := new(errgroup.Group)
		errG.Go(func() error {
			return bp.removeTableEntry(ctx, processedCh)
		})
		if err := errG.Wait(); err != nil {
			finalErrCh <- []error{err}
		}
	}()
	// Process errors from slot processing.
	go bp.handleProcessingErrors(ctx, errCh)

	// Get slots from the DB.
	go func() {
		errs := bp.getSlotRange(ctx, slotsCh, minimumSlot) // Periodically adds new entries....
		if errs != nil {
			finalErrCh <- errs
		}
		finalErrCh <- nil
		log.Debug("We are stopping the processing of adding new entries")
	}()
	log.Debug("Waiting for shutdown signal from channel")
	select {
	case <-ctx.Done():
		log.Debug("Received shutdown signal from channel")
		return nil
	case errs := <-finalErrCh:
		log.Debug("Finishing the batchProcess")
		return errs
	}
}
