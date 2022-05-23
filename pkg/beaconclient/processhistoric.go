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
	// Get a single highest priority and non-checked out row.
	getBpEntryStmt string = `SELECT start_slot, end_slot FROM ethcl.historic_process
	WHERE checked_out=false
	ORDER BY priority ASC
	LIMIT 1;`
	lockBpEntryStmt string = `UPDATE ethcl.historic_process
	SET checked_out=true
	WHERE start_slot=$1 AND end_slot=$2;`
	deleteSlotsEntryStmt string = `DELETE FROM ethcl.historic_process
	WHERE start_slot=$1 AND end_slot=$2;`
)

type historicProcessing struct {
	db      sql.Database
	metrics *BeaconClientMetrics
}

// Get a single row of historical slots from the table.
func (hp historicProcessing) getSlotRange(slotCh chan<- slotsToProcess) []error {
	return getBatchProcessRow(hp.db, getBpEntryStmt, lockBpEntryStmt, slotCh)
}

// Remove the table entry.
func (hp historicProcessing) removeTableEntry(processCh <-chan slotsToProcess) error {
	return removeRowPostProcess(hp.db, processCh, deleteSlotsEntryStmt)
}

// Remove the table entry.
func (hp historicProcessing) handleProcessingErrors(errMessages <-chan batchHistoricError) {
	for {
		errMs := <-errMessages
		writeKnownGaps(hp.db, 1, errMs.slot, errMs.slot, errMs.err, errMs.errProcess, hp.metrics)
	}
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
func getBatchProcessRow(db sql.Database, getStartEndSlotStmt string, checkOutRowStmt string, slotCh chan<- slotsToProcess) []error {
	errCount := make([]error, 0)

	for len(errCount) < 5 {
		ctx := context.Background()

		// Setup TX
		tx, err := db.Begin(ctx)
		if err != nil {
			errCount = append(errCount, err)
			continue
		}
		defer tx.Rollback(ctx)

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
		res, err := tx.Exec(ctx, checkOutRowStmt, sp.startSlot, sp.endSlot)
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
		if rows != 1 {
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
	return errCount
}

// After a row has been processed it should be removed from its appropriate table.
func removeRowPostProcess(db sql.Database, processCh <-chan slotsToProcess, removeStmt string) error {
	for {
		slots := <-processCh
		_, err := db.Exec(context.Background(), removeStmt, strconv.Itoa(slots.startSlot), slots.endSlot)
		if err != nil {
			return err
		}
	}
}
