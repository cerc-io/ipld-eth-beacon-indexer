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
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

var (
	// Get a single non-checked out row row from ethcl.known_gaps.
	getKgEntryStmt string = `SELECT start_slot, end_slot FROM ethcl.known_gaps
	WHERE checked_out=false
	LIMIT 1;`
	// Used to periodically check to see if there is a new entry in the ethcl.known_gaps table.
	checkKgEntryStmt string = `SELECT * FROM ethcl.known_gaps WHERE checked_out=false;`
	// Used to checkout a row from the ethcl.known_gaps table
	lockKgEntryStmt string = `UPDATE ethcl.known_gaps
	SET checked_out=true, checked_out_by=$3
	WHERE start_slot=$1 AND end_slot=$2;`
	// Used to delete an entry from the knownGaps table
	deleteKgEntryStmt string = `DELETE FROM ethcl.known_gaps
	WHERE start_slot=$1 AND end_slot=$2;`
	// Used to check to see if a single slot exists in the known_gaps table.
	checkKgSingleSlotStmt string = `SELECT start_slot, end_slot FROM ethcl.known_gaps
	WHERE start_slot=$1 AND end_slot=$2;`
	// Used to update every single row that this node has checked out.
	releaseKgLockStmt string = `UPDATE ethcl.known_gaps
	SET checked_out=false, checked_out_by=null
	WHERE checked_out_by=$1`
)

type KnownGapsProcessing struct {
	db                   sql.Database         //db connection
	metrics              *BeaconClientMetrics // metrics for beaconclient
	uniqueNodeIdentifier int                  // node unique identifier.
}

// This function will perform all the heavy lifting for tracking the head of the chain.
func (bc *BeaconClient) ProcessKnownGaps(ctx context.Context, maxWorkers int) []error {
	log.Info("We are starting the known gaps processing service.")
	bc.KnownGapsProcess = KnownGapsProcessing{db: bc.Db, uniqueNodeIdentifier: bc.UniqueNodeIdentifier, metrics: bc.Metrics}
	errs := handleBatchProcess(ctx, maxWorkers, bc.KnownGapsProcess, bc.KnownGapsProcess.db, bc.ServerEndpoint, bc.Metrics, bc.CheckDb)
	log.Debug("Exiting known gaps processing service")
	return errs
}

// This function will perform all the necessary clean up tasks for stopping historical processing.
func (bc *BeaconClient) StopKnownGapsProcessing(cancel context.CancelFunc) error {
	log.Info("We are stopping the known gaps processing service.")
	err := bc.KnownGapsProcess.releaseDbLocks(cancel)
	if err != nil {
		loghelper.LogError(err).WithField("uniqueIdentifier", bc.UniqueNodeIdentifier).Error("We were unable to remove the locks from the ethcl.known_gaps table. Manual Intervention is needed!")
	}
	return nil
}

// Get a single row of historical slots from the table.
func (kgp KnownGapsProcessing) getSlotRange(ctx context.Context, slotCh chan<- slotsToProcess) []error {
	return getBatchProcessRow(ctx, kgp.db, getKgEntryStmt, checkKgEntryStmt, lockKgEntryStmt, slotCh, strconv.Itoa(kgp.uniqueNodeIdentifier))
}

// Remove the table entry.
func (kgp KnownGapsProcessing) removeTableEntry(ctx context.Context, processCh <-chan slotsToProcess) error {
	return removeRowPostProcess(ctx, kgp.db, processCh, QueryBySlotStmt, deleteKgEntryStmt)
}

// Remove the table entry.
func (kgp KnownGapsProcessing) handleProcessingErrors(ctx context.Context, errMessages <-chan batchHistoricError) {
	for {
		select {
		case <-ctx.Done():
			return
		case errMs := <-errMessages:
			// Check to see if this if this entry already exists.
			res, err := kgp.db.Exec(context.Background(), checkKgSingleSlotStmt, errMs.slot, errMs.slot)
			if err != nil {
				loghelper.LogSlotError(strconv.Itoa(errMs.slot), err).Error("Unable to see if this slot is in the ethcl.known_gaps table")
			}

			rows, err := res.RowsAffected()
			if err != nil {
				loghelper.LogSlotError(strconv.Itoa(errMs.slot), err).WithFields(log.Fields{
					"queryStatement": checkKgSingleSlotStmt,
				}).Error("Unable to get the number of rows affected by this statement.")
			}

			if rows > 0 {
				loghelper.LogSlotError(strconv.Itoa(errMs.slot), errMs.err).Error("We received an error when processing a knownGap")
				err = updateKnownGapErrors(kgp.db, errMs.slot, errMs.slot, errMs.err, kgp.metrics)
				if err != nil {
					loghelper.LogSlotError(strconv.Itoa(errMs.slot), err).Error("Error processing known gap")
				}
			} else {
				writeKnownGaps(kgp.db, 1, errMs.slot, errMs.slot, errMs.err, errMs.errProcess, kgp.metrics)
			}
		}
	}

}

// Updated checked_out column for the uniqueNodeIdentifier.
func (kgp KnownGapsProcessing) releaseDbLocks(cancel context.CancelFunc) error {
	go func() { cancel() }()
	log.Debug("Updating all the entries to ethcl.known_gaps")
	log.Debug("Db: ", kgp.db)
	log.Debug("kgp.uniqueNodeIdentifier ", kgp.uniqueNodeIdentifier)
	res, err := kgp.db.Exec(context.Background(), releaseKgLockStmt, kgp.uniqueNodeIdentifier)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	log.WithField("rowCount", rows).Info("Released knownGaps locks for specified rows.")
	return nil
}
