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
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

var (
	// Get a single highest priority and non-checked out row row from ethcl.historical_process
	getHpEntryStmt string = `SELECT start_slot, end_slot FROM ethcl.historic_process
	WHERE checked_out=false
	ORDER BY priority ASC
	LIMIT 1;`
	// Used to periodically check to see if there is a new entry in the ethcl.historic_process table.
	checkHpEntryStmt string = `SELECT * FROM ethcl.historic_process WHERE checked_out=false;`
	// Used to checkout a row from the ethcl.historic_process table
	lockHpEntryStmt string = `UPDATE ethcl.historic_process
	SET checked_out=true, checked_out_by=$3
	WHERE start_slot=$1 AND end_slot=$2;`
	// Used to delete an entry from the ethcl.historic_process table
	deleteHpEntryStmt string = `DELETE FROM ethcl.historic_process
	WHERE start_slot=$1 AND end_slot=$2;`
	// Used to update every single row that this node has checked out.
	releaseHpLockStmt string = `UPDATE ethcl.historic_process
	SET checked_out=false
	WHERE checked_out_by=$1`
)

type historicProcessing struct {
	db                   sql.Database         //db connection
	metrics              *BeaconClientMetrics // metrics for beaconclient
	uniqueNodeIdentifier int                  // node unique identifier.
	finishProcessing     chan int             // A channel which indicates to the process handleBatchProcess function that its time to end.
}

// Get a single row of historical slots from the table.
func (hp historicProcessing) getSlotRange(slotCh chan<- slotsToProcess) []error {
	return getBatchProcessRow(hp.db, getHpEntryStmt, checkHpEntryStmt, lockHpEntryStmt, slotCh, strconv.Itoa(hp.uniqueNodeIdentifier))
}

// Remove the table entry.
func (hp historicProcessing) removeTableEntry(processCh <-chan slotsToProcess) error {
	return removeRowPostProcess(hp.db, processCh, QueryBySlotStmt, deleteHpEntryStmt)
}

// Remove the table entry.
func (hp historicProcessing) handleProcessingErrors(errMessages <-chan batchHistoricError) {
	for {
		errMs := <-errMessages
		loghelper.LogSlotError(strconv.Itoa(errMs.slot), errMs.err)
		writeKnownGaps(hp.db, 1, errMs.slot, errMs.slot, errMs.err, errMs.errProcess, hp.metrics)
	}
}

func (hp historicProcessing) releaseDbLocks() error {
	go func() { hp.finishProcessing <- 1 }()
	log.Debug("Updating all the entries to ethcl.historical processing")
	res, err := hp.db.Exec(context.Background(), releaseHpLockStmt, hp.uniqueNodeIdentifier)
	if err != nil {
		return fmt.Errorf("Unable to remove lock from ethcl.historical_processing table for node %d, error is %e", hp.uniqueNodeIdentifier, err)
	}
	log.Debug("Update all the entries to ethcl.historical processing")
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("Unable to calculated number of rows affected by releasing locks from ethcl.historical_processing table for node %d, error is %e", hp.uniqueNodeIdentifier, err)
	}
	log.WithField("rowCount", rows).Info("Released historicalProcess locks for specified rows.")
	return nil
}

// Process the slot range.
func processSlotRangeWorker(workCh <-chan int, errCh chan<- batchHistoricError, db sql.Database, serverAddress string, metrics *BeaconClientMetrics) {
	for slot := range workCh {
		log.Debug("Handling slot: ", slot)
		err, errProcess := handleHistoricSlot(db, serverAddress, slot, metrics)
		errMs := batchHistoricError{
			err:        err,
			errProcess: errProcess,
			slot:       slot,
		}
		if err != nil {
			errCh <- errMs
		}
	}
}

// A wrapper function that insert the start_slot and end_slot from a single row into a channel.
// It also locks the row by updating the checked_out column.
// The statement for getting the start_slot and end_slot must be provided.
// The statement for "locking" the row must also be provided.
func getBatchProcessRow(db sql.Database, getStartEndSlotStmt string, checkNewRowsStmt string, checkOutRowStmt string, slotCh chan<- slotsToProcess, uniqueNodeIdentifier string) []error {
	errCount := make([]error, 0)

	// 5 is an arbitrary number. It allows us to retry a few times before
	// ending the application.
	prevErrCount := 0
	for len(errCount) < 5 {
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
			time.Sleep(1000 * time.Millisecond)
			log.Debug("We are checking rows, be patient")
			continue
		}
		log.Debug("We found a new row")
		ctx := context.Background()

		// Setup TX
		tx, err := db.Begin(ctx)
		if err != nil {
			loghelper.LogError(err).Error("We are unable to Begin a SQL transaction")
			errCount = append(errCount, err)
			continue
		}
		defer func() {
			err := tx.Rollback(ctx)
			if err != nil && err != pgx.ErrTxClosed {
				loghelper.LogError(err).Error("We were unable to Rollback a transaction")
				errCount = append(errCount, err)
			}
		}()

		// Query the DB for slots.
		sp := slotsToProcess{}
		err = tx.QueryRow(ctx, getStartEndSlotStmt).Scan(&sp.startSlot, &sp.endSlot)
		if err != nil {
			if err == pgx.ErrNoRows {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			loghelper.LogSlotRangeStatementError(strconv.Itoa(sp.startSlot), strconv.Itoa(sp.endSlot), getStartEndSlotStmt, err).Error("Unable to get a row")
			errCount = append(errCount, err)
			continue
		}

		// Checkout the Row
		res, err := tx.Exec(ctx, checkOutRowStmt, sp.startSlot, sp.endSlot, uniqueNodeIdentifier)
		if err != nil {
			loghelper.LogSlotRangeStatementError(strconv.Itoa(sp.startSlot), strconv.Itoa(sp.endSlot), checkOutRowStmt, err).Error("Unable to checkout the row")
			errCount = append(errCount, err)
			continue
		}
		rows, err := res.RowsAffected()
		if err != nil {
			loghelper.LogSlotRangeStatementError(strconv.Itoa(sp.startSlot), strconv.Itoa(sp.endSlot), checkOutRowStmt, fmt.Errorf("Unable to determine the rows affected when trying to checkout a row."))
			errCount = append(errCount, err)
			continue
		}
		if rows > 1 {
			loghelper.LogSlotRangeStatementError(strconv.Itoa(sp.startSlot), strconv.Itoa(sp.endSlot), checkOutRowStmt, err).WithFields(log.Fields{
				"rowsReturn": rows,
			}).Error("We locked too many rows.....")
			errCount = append(errCount, err)
			continue
		}
		if rows == 0 {
			loghelper.LogSlotRangeStatementError(strconv.Itoa(sp.startSlot), strconv.Itoa(sp.endSlot), checkOutRowStmt, err).WithFields(log.Fields{
				"rowsReturn": rows,
			}).Error("We did not lock a single row.")
			errCount = append(errCount, err)
			continue
		}
		err = tx.Commit(ctx)
		if err != nil {
			loghelper.LogSlotRangeError(strconv.Itoa(sp.startSlot), strconv.Itoa(sp.endSlot), err).Error("Unable commit transactions.")
			errCount = append(errCount, err)
			continue
		}
		slotCh <- sp
	}
	log.WithFields(log.Fields{
		"ErrCount": errCount,
	}).Error("The ErrCounter")
	return errCount
}

// After a row has been processed it should be removed from its appropriate table.
func removeRowPostProcess(db sql.Database, processCh <-chan slotsToProcess, checkProcessedStmt, removeStmt string) error {
	errCh := make(chan error)
	for {
		slots := <-processCh
		// Make sure the start and end slot exist in the slots table.
		go func() {
			finishedProcess := false
			for !finishedProcess {
				isStartProcess, err := isSlotProcessed(db, checkProcessedStmt, strconv.Itoa(slots.startSlot))
				if err != nil {
					errCh <- err
				}
				isEndProcess, err := isSlotProcessed(db, checkProcessedStmt, strconv.Itoa(slots.endSlot))
				if err != nil {
					errCh <- err
				}
				if isStartProcess && isEndProcess {
					finishedProcess = true
				}
			}

			_, err := db.Exec(context.Background(), removeStmt, strconv.Itoa(slots.startSlot), strconv.Itoa(slots.endSlot))
			if err != nil {
				errCh <- err
			}

		}()
		if len(errCh) != 0 {
			return <-errCh
		}
	}
}
