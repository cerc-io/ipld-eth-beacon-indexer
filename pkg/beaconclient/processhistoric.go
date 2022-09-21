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

// This file contains all the code to process historic slots.

package beaconclient

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v4"
	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/loghelper"
)

var (
	// Get a single highest priority and non-checked out row row from eth_beacon.historical_process
	getHpEntryStmt string = `SELECT start_slot, end_slot FROM eth_beacon.historic_process
	WHERE checked_out=false
	ORDER BY priority ASC
	LIMIT 1;`
	// Used to periodically check to see if there is a new entry in the eth_beacon.historic_process table.
	checkHpEntryStmt string = `SELECT * FROM eth_beacon.historic_process WHERE checked_out=false;`
	// Used to checkout a row from the eth_beacon.historic_process table
	lockHpEntryStmt string = `UPDATE eth_beacon.historic_process
	SET checked_out=true, checked_out_by=$3
	WHERE start_slot=$1 AND end_slot=$2;`
	// Used to delete an entry from the eth_beacon.historic_process table
	deleteHpEntryStmt string = `DELETE FROM eth_beacon.historic_process
	WHERE start_slot=$1 AND end_slot=$2;`
	// Used to update every single row that this node has checked out.
	releaseHpLockStmt string = `UPDATE eth_beacon.historic_process
	SET checked_out=false, checked_out_by=null
	WHERE checked_out_by=$1`
)

type HistoricProcessing struct {
	db                   sql.Database         //db connection
	metrics              *BeaconClientMetrics // metrics for beaconclient
	uniqueNodeIdentifier int                  // node unique identifier.
}

// Get a single row of historical slots from the table.
func (hp HistoricProcessing) getSlotRange(ctx context.Context, slotCh chan<- slotsToProcess) []error {
	return getBatchProcessRow(ctx, hp.db, getHpEntryStmt, checkHpEntryStmt, lockHpEntryStmt, slotCh, strconv.Itoa(hp.uniqueNodeIdentifier))
}

// Remove the table entry.
func (hp HistoricProcessing) removeTableEntry(ctx context.Context, processCh <-chan slotsToProcess) error {
	return removeRowPostProcess(ctx, hp.db, processCh, QueryBySlotStmt, deleteHpEntryStmt)
}

// Remove the table entry.
func (hp HistoricProcessing) handleProcessingErrors(ctx context.Context, errMessages <-chan batchHistoricError) {
	for {
		select {
		case <-ctx.Done():
			return
		case errMs := <-errMessages:
			loghelper.LogSlotError(strconv.FormatUint(errMs.slot, 10), errMs.err)
			writeKnownGaps(hp.db, 1, errMs.slot, errMs.slot, errMs.err, errMs.errProcess, hp.metrics)
		}
	}
}

// "un"-checkout the rows held by this DB in the eth_beacon.historical_process table.
func (hp HistoricProcessing) releaseDbLocks() error {
	log.Debug("Updating all the entries to eth_beacon.historical processing")
	res, err := hp.db.Exec(context.Background(), releaseHpLockStmt, hp.uniqueNodeIdentifier)
	if err != nil {
		return fmt.Errorf("Unable to remove lock from eth_beacon.historical_processing table for node %d, error is %e", hp.uniqueNodeIdentifier, err)
	}
	log.Debug("Update all the entries to eth_beacon.historical processing")
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("Unable to calculated number of rows affected by releasing locks from eth_beacon.historical_processing table for node %d, error is %e", hp.uniqueNodeIdentifier, err)
	}
	log.WithField("rowCount", rows).Info("Released historicalProcess locks for specified rows.")
	return nil
}

// Process the slot range.
func processSlotRangeWorker(ctx context.Context, workCh <-chan uint64, errCh chan<- batchHistoricError, spd SlotProcessingDetails, incrementTracker func(uint64)) {
	for {
		select {
		case <-ctx.Done():
			return
		case slot := <-workCh:
			log.Debug("Handling slot: ", slot)
			err, errProcess := handleHistoricSlot(ctx, slot, spd)
			if err != nil {
				errMs := batchHistoricError{
					err:        err,
					errProcess: errProcess,
					slot:       slot,
				}
				errCh <- errMs
			} else {
				incrementTracker(1)
			}
		}
	}
}

// A wrapper function that insert the start_slot and end_slot from a single row into a channel.
// It also locks the row by updating the checked_out column.
// The statement for getting the start_slot and end_slot must be provided.
// The statement for "locking" the row must also be provided.
func getBatchProcessRow(ctx context.Context, db sql.Database, getStartEndSlotStmt string, checkNewRowsStmt string, checkOutRowStmt string, slotCh chan<- slotsToProcess, uniqueNodeIdentifier string) []error {
	errCount := make([]error, 0)

	// 5 is an arbitrary number. It allows us to retry a few times before
	// ending the application.
	prevErrCount := 0
	for len(errCount) < 5 {
		select {
		case <-ctx.Done():
			return errCount
		default:
			if len(errCount) != prevErrCount {
				log.WithFields(log.Fields{
					"errCount": errCount,
				}).Error("New error entry added")
			}
			processRow, err := db.Exec(context.Background(), checkNewRowsStmt)
			if err != nil {
				errCount = append(errCount, err)
			}
			row, err := processRow.RowsAffected()
			if err != nil {
				errCount = append(errCount, err)
			}
			if row < 1 {
				time.Sleep(3 * time.Second)
				log.Debug("We are checking rows, be patient")
				break
			}
			log.Debug("We found a new row")
			dbCtx := context.Background()

			// Setup TX
			tx, err := db.Begin(dbCtx)
			if err != nil {
				loghelper.LogError(err).Error("We are unable to Begin a SQL transaction")
				errCount = append(errCount, err)
				break
			}
			defer func() {
				err := tx.Rollback(dbCtx)
				if err != nil && err != pgx.ErrTxClosed {
					loghelper.LogError(err).Error("We were unable to Rollback a transaction")
					errCount = append(errCount, err)
				}
			}()

			// Query the DB for slots.
			sp := slotsToProcess{}
			err = tx.QueryRow(dbCtx, getStartEndSlotStmt).Scan(&sp.startSlot, &sp.endSlot)
			if err != nil {
				if err == pgx.ErrNoRows {
					time.Sleep(1 * time.Second)
					break
				}
				loghelper.LogSlotRangeStatementError(strconv.FormatUint(sp.startSlot, 10), strconv.FormatUint(sp.endSlot, 10), getStartEndSlotStmt, err).Error("Unable to get a row")
				errCount = append(errCount, err)
				break
			}

			// Checkout the Row
			res, err := tx.Exec(dbCtx, checkOutRowStmt, sp.startSlot, sp.endSlot, uniqueNodeIdentifier)
			if err != nil {
				loghelper.LogSlotRangeStatementError(strconv.FormatUint(sp.startSlot, 10), strconv.FormatUint(sp.endSlot, 10), checkOutRowStmt, err).Error("Unable to checkout the row")
				errCount = append(errCount, err)
				break
			}
			rows, err := res.RowsAffected()
			if err != nil {
				loghelper.LogSlotRangeStatementError(strconv.FormatUint(sp.startSlot, 10), strconv.FormatUint(sp.endSlot, 10), checkOutRowStmt, fmt.Errorf("Unable to determine the rows affected when trying to checkout a row."))
				errCount = append(errCount, err)
				break
			}
			if rows > 1 {
				loghelper.LogSlotRangeStatementError(strconv.FormatUint(sp.startSlot, 10), strconv.FormatUint(sp.endSlot, 10), checkOutRowStmt, err).WithFields(log.Fields{
					"rowsReturn": rows,
				}).Error("We locked too many rows.....")
				errCount = append(errCount, err)
				break
			}
			if rows == 0 {
				loghelper.LogSlotRangeStatementError(strconv.FormatUint(sp.startSlot, 10), strconv.FormatUint(sp.endSlot, 10), checkOutRowStmt, err).WithFields(log.Fields{
					"rowsReturn": rows,
				}).Error("We did not lock a single row.")
				errCount = append(errCount, err)
				break
			}
			err = tx.Commit(dbCtx)
			if err != nil {
				loghelper.LogSlotRangeError(strconv.FormatUint(sp.startSlot, 10), strconv.FormatUint(sp.endSlot, 10), err).Error("Unable commit transactions.")
				errCount = append(errCount, err)
				break
			}
			log.WithField("slots", sp).Debug("Added a new slots to be processed")
			slotCh <- sp
		}
	}
	log.WithFields(log.Fields{
		"ErrCount": errCount,
	}).Error("The ErrCounter")
	return errCount
}

// After a row has been processed it should be removed from its appropriate table.
func removeRowPostProcess(ctx context.Context, db sql.Database, processCh <-chan slotsToProcess, checkProcessedStmt, removeStmt string) error {
	errCh := make(chan error)
	for {
		select {
		case <-ctx.Done():
			return nil
		case slots := <-processCh:
			// Make sure the start and end slot exist in the slots table.
			go func() {
				log.WithFields(log.Fields{
					"startSlot": slots.startSlot,
					"endSlot":   slots.endSlot,
				}).Debug("Starting to check to see if the following slots have been processed")
				for {
					isStartProcess, err := isSlotProcessed(db, checkProcessedStmt, strconv.FormatUint(slots.startSlot, 10))
					if err != nil {
						errCh <- err
					}
					isEndProcess, err := isSlotProcessed(db, checkProcessedStmt, strconv.FormatUint(slots.endSlot, 10))
					if err != nil {
						errCh <- err
					}
					if isStartProcess && isEndProcess {
						break
					}
					time.Sleep(3 * time.Second)
				}

				_, err := db.Exec(context.Background(), removeStmt, strconv.FormatUint(slots.startSlot, 10), strconv.FormatUint(slots.endSlot, 10))
				if err != nil {
					errCh <- err
				}

			}()
			if len(errCh) != 0 {
				return <-errCh
			}
		}
	}
}
