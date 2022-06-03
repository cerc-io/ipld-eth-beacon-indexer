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
package beaconclient

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v4"
	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
	"golang.org/x/sync/errgroup"
)

var (
	// Statement to upsert to the ethcl.slots table.
	UpsertSlotsStmt string = `
INSERT INTO ethcl.slots (epoch, slot, block_root, state_root, status)
VALUES ($1, $2, $3, $4, $5) ON CONFLICT (slot, block_root) DO NOTHING`
	// Statement to upsert to the ethcl.signed_beacon_blocks table.
	UpsertSignedBeaconBlockStmt string = `
INSERT INTO ethcl.signed_beacon_block (slot, block_root, parent_block_root, eth1_block_hash, mh_key)
VALUES ($1, $2, $3, $4, $5) ON CONFLICT (slot, block_root) DO NOTHING`
	// Statement to upsert to the ethcl.beacon_state table.
	UpsertBeaconState string = `
INSERT INTO ethcl.beacon_state (slot, state_root, mh_key)
VALUES ($1, $2, $3) ON CONFLICT (slot, state_root) DO NOTHING`
	// Statement to upsert to the public.blocks table.
	UpsertBlocksStmt string = `
INSERT INTO public.blocks (key, data)
VALUES ($1, $2) ON CONFLICT (key) DO NOTHING`
	UpdateForkedStmt string = `UPDATE ethcl.slots
	SET status='forked'
	WHERE slot=$1 AND block_root<>$2
	RETURNING block_root;`
	UpdateProposedStmt string = `UPDATE ethcl.slots
	SET status='proposed'
	WHERE slot=$1 AND block_root=$2
	RETURNING block_root;`
	CheckProposedStmt string = `SELECT slot, block_root
	FROM ethcl.slots
	WHERE slot=$1 AND block_root=$2;`
	// Check to see if the slot and block_root exist in ethcl.signed_beacon_block
	CheckSignedBeaconBlockStmt string = `SELECT slot, block_root
	FROM ethcl.signed_beacon_block
	WHERE slot=$1 AND block_root=$2`
	// Check to see if the slot and state_root exist in ethcl.beacon_state
	CheckBeaconStateStmt string = `SELECT slot, state_root
	FROM ethcl.beacon_state
	WHERE slot=$1 AND state_root=$2`
	// Used to get a single slot from the table if it exists
	QueryBySlotStmt string = `SELECT slot
	FROM ethcl.slots
	WHERE slot=$1`
	// Statement to insert known_gaps. We don't pass in timestamp, we let the server take care of that one.
	UpsertKnownGapsStmt string = `
INSERT INTO ethcl.known_gaps (start_slot, end_slot, checked_out, reprocessing_error, entry_error, entry_process)
VALUES ($1, $2, $3, $4, $5, $6) on CONFLICT (start_slot, end_slot) DO NOTHING`
	UpsertKnownGapsErrorStmt string = `
	UPDATE ethcl.known_gaps
	SET reprocessing_error=$3
	WHERE start_slot=$1 AND end_slot=$2;`
	// Get the highest slot if one exists
	QueryHighestSlotStmt string = "SELECT COALESCE(MAX(slot), 0) FROM ethcl.slots"
)

// Put all functionality to prepare the write object
// And write it in this file.
// Remove any of it from the processslot file.
type DatabaseWriter struct {
	Db                   sql.Database
	Tx                   sql.Tx
	Ctx                  context.Context
	Metrics              *BeaconClientMetrics
	DbSlots              *DbSlots
	DbSignedBeaconBlock  *DbSignedBeaconBlock
	DbBeaconState        *DbBeaconState
	rawBeaconState       []byte
	rawSignedBeaconBlock []byte
}

func CreateDatabaseWrite(db sql.Database, slot int, stateRoot string, blockRoot string, parentBlockRoot string,
	eth1BlockHash string, status string, rawSignedBeaconBlock []byte, rawBeaconState []byte, metrics *BeaconClientMetrics) (*DatabaseWriter, error) {
	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		loghelper.LogError(err).Error("We are unable to Begin a SQL transaction")
	}
	dw := &DatabaseWriter{
		Db:                   db,
		Tx:                   tx,
		Ctx:                  ctx,
		rawBeaconState:       rawBeaconState,
		rawSignedBeaconBlock: rawSignedBeaconBlock,
		Metrics:              metrics,
	}
	dw.prepareSlotsModel(slot, stateRoot, blockRoot, status)
	err = dw.prepareSignedBeaconBlockModel(slot, blockRoot, parentBlockRoot, eth1BlockHash)
	if err != nil {
		return nil, err
	}
	err = dw.prepareBeaconStateModel(slot, stateRoot)
	if err != nil {
		return nil, err
	}
	return dw, err
}

// Write functions to write each all together...
// Should I do one atomic write?
// Create the model for the ethcl.slots table
func (dw *DatabaseWriter) prepareSlotsModel(slot int, stateRoot string, blockRoot string, status string) {
	dw.DbSlots = &DbSlots{
		Epoch:     calculateEpoch(slot, bcSlotsPerEpoch),
		Slot:      strconv.Itoa(slot),
		StateRoot: stateRoot,
		BlockRoot: blockRoot,
		Status:    status,
	}
	log.Debug("dw.DbSlots: ", dw.DbSlots)

}

// Create the model for the ethcl.signed_beacon_block table.
func (dw *DatabaseWriter) prepareSignedBeaconBlockModel(slot int, blockRoot string, parentBlockRoot string, eth1BlockHash string) error {
	mhKey, err := MultihashKeyFromSSZRoot([]byte(dw.DbSlots.BlockRoot))
	if err != nil {
		return err
	}
	dw.DbSignedBeaconBlock = &DbSignedBeaconBlock{
		Slot:          strconv.Itoa(slot),
		BlockRoot:     blockRoot,
		ParentBlock:   parentBlockRoot,
		Eth1BlockHash: eth1BlockHash,
		MhKey:         mhKey,
	}
	log.Debug("dw.DbSignedBeaconBlock: ", dw.DbSignedBeaconBlock)
	return nil
}

// Create the model for the ethcl.beacon_state table.
func (dw *DatabaseWriter) prepareBeaconStateModel(slot int, stateRoot string) error {
	mhKey, err := MultihashKeyFromSSZRoot([]byte(dw.DbSlots.StateRoot))
	if err != nil {
		return err
	}
	dw.DbBeaconState = &DbBeaconState{
		Slot:      strconv.Itoa(slot),
		StateRoot: stateRoot,
		MhKey:     mhKey,
	}
	log.Debug("dw.DbBeaconState: ", dw.DbBeaconState)
	return nil
}

// Add all the data for a given slot to a SQL transaction.
// Originally it wrote to each table individually.
func (dw *DatabaseWriter) transactFullSlot() error {
	// If an error occurs, write to knownGaps table.
	log.WithFields(log.Fields{
		"slot": dw.DbSlots.Slot,
	}).Debug("Starting to write to the DB.")
	err := dw.transactSlots()
	if err != nil {
		loghelper.LogSlotError(dw.DbSlots.Slot, err).Error("We couldn't write to the ethcl.slots table...")
		return err
	}
	log.Debug("We finished writing to the ethcl.slots table.")
	if dw.DbSlots.Status != "skipped" {
		//errG, _ := errgroup.WithContext(context.Background())
		//errG.Go(func() error {
		//	return dw.transactSignedBeaconBlocks()
		//})
		//errG.Go(func() error {
		//	return dw.transactBeaconState()
		//})
		//if err := errG.Wait(); err != nil {
		//	loghelper.LogSlotError(dw.DbSlots.Slot, err).Error("We couldn't write to the ethcl block or state table...")
		//	return err
		//}
		// Might want to seperate writing to public.blocks so we can do this concurrently...
		err := dw.transactSignedBeaconBlocks()
		if err != nil {
			loghelper.LogSlotError(dw.DbSlots.Slot, err).Error("We couldn't write to the ethcl block table...")
			return err
		}
		err = dw.transactBeaconState()
		if err != nil {
			loghelper.LogSlotError(dw.DbSlots.Slot, err).Error("We couldn't write to the ethcl state table...")
			return err
		}
	}
	dw.Metrics.IncrementSlotInserts(1)
	return nil
}

// Add data for the ethcl.slots table to a transaction. For now this is only one function.
// But in the future if we need to incorporate any FK's or perform any actions to write to the
// slots table we can do it all here.
func (dw *DatabaseWriter) transactSlots() error {
	return dw.upsertSlots()
}

// Upsert to the ethcl.slots table.
func (dw *DatabaseWriter) upsertSlots() error {
	_, err := dw.Tx.Exec(dw.Ctx, UpsertSlotsStmt, dw.DbSlots.Epoch, dw.DbSlots.Slot, dw.DbSlots.BlockRoot, dw.DbSlots.StateRoot, dw.DbSlots.Status)
	if err != nil {
		loghelper.LogSlotError(dw.DbSlots.Slot, err).Error("Unable to write to the slot to the ethcl.slots table")
		return err
	}
	return nil
}

// Add the information for the signed_beacon_block to a transaction.
func (dw *DatabaseWriter) transactSignedBeaconBlocks() error {
	err := dw.upsertPublicBlocks(dw.DbSignedBeaconBlock.MhKey, dw.rawSignedBeaconBlock)
	if err != nil {
		return err
	}
	err = dw.upsertSignedBeaconBlock()
	if err != nil {
		return err
	}
	return nil
}

// Upsert to public.blocks.
func (dw *DatabaseWriter) upsertPublicBlocks(key string, data []byte) error {
	_, err := dw.Tx.Exec(dw.Ctx, UpsertBlocksStmt, key, data)
	if err != nil {
		loghelper.LogSlotError(dw.DbSlots.Slot, err).Error("Unable to write to the slot to the public.blocks table")
		return err
	}
	return nil
}

// Upsert to the ethcl.signed_beacon_block table.
func (dw *DatabaseWriter) upsertSignedBeaconBlock() error {
	_, err := dw.Tx.Exec(dw.Ctx, UpsertSignedBeaconBlockStmt, dw.DbSignedBeaconBlock.Slot, dw.DbSignedBeaconBlock.BlockRoot, dw.DbSignedBeaconBlock.ParentBlock, dw.DbSignedBeaconBlock.Eth1BlockHash, dw.DbSignedBeaconBlock.MhKey)
	if err != nil {
		loghelper.LogSlotError(dw.DbSlots.Slot, err).WithFields(log.Fields{"block_root": dw.DbSignedBeaconBlock.BlockRoot}).Error("Unable to write to the slot to the ethcl.signed_beacon_block table")
		return err
	}
	return nil
}

// Add the information for the beacon_state to a transaction.
func (dw *DatabaseWriter) transactBeaconState() error {
	err := dw.upsertPublicBlocks(dw.DbBeaconState.MhKey, dw.rawBeaconState)
	if err != nil {
		return err
	}
	err = dw.upsertBeaconState()
	if err != nil {
		return err
	}
	return nil
}

// Upsert to the ethcl.beacon_state table.
func (dw *DatabaseWriter) upsertBeaconState() error {
	_, err := dw.Tx.Exec(dw.Ctx, UpsertBeaconState, dw.DbBeaconState.Slot, dw.DbBeaconState.StateRoot, dw.DbBeaconState.MhKey)
	if err != nil {
		loghelper.LogSlotError(dw.DbSlots.Slot, err).Error("Unable to write to the slot to the ethcl.beacon_state table")
		return err
	}
	return nil
}

// Update a given slot to be marked as forked within a transaction. Provide the slot and the latest latestBlockRoot.
// We will mark all entries for the given slot that don't match the provided latestBlockRoot as forked.
func transactReorgs(tx sql.Tx, ctx context.Context, slot string, latestBlockRoot string, metrics *BeaconClientMetrics) {
	slotNum, strErr := strconv.Atoi(slot)
	if strErr != nil {
		loghelper.LogReorgError(slot, latestBlockRoot, strErr).Error("We can't convert the slot to an int...")
	}

	forkCount, err := updateForked(tx, ctx, slot, latestBlockRoot)
	if err != nil {
		loghelper.LogReorgError(slot, latestBlockRoot, err).Error("We ran into some trouble while updating all forks.")
		transactKnownGaps(tx, ctx, 1, slotNum, slotNum, err, "reorg", metrics)
	}
	proposedCount, err := updateProposed(tx, ctx, slot, latestBlockRoot)
	if err != nil {
		loghelper.LogReorgError(slot, latestBlockRoot, err).Error("We ran into some trouble while trying to update the proposed slot.")
		transactKnownGaps(tx, ctx, 1, slotNum, slotNum, err, "reorg", metrics)
	}

	if forkCount > 0 {
		loghelper.LogReorg(slot, latestBlockRoot).WithFields(log.Fields{
			"forkCount": forkCount,
		}).Info("Updated rows that were forked.")
	} else {
		loghelper.LogReorg(slot, latestBlockRoot).WithFields(log.Fields{
			"forkCount": forkCount,
		}).Warn("There were no forked rows to update.")
	}

	if proposedCount == 1 {
		loghelper.LogReorg(slot, latestBlockRoot).WithFields(log.Fields{
			"proposedCount": proposedCount,
		}).Info("Updated the row that should have been marked as proposed.")
	} else if proposedCount > 1 {
		loghelper.LogReorg(slot, latestBlockRoot).WithFields(log.Fields{
			"proposedCount": proposedCount,
		}).Error("Too many rows were marked as proposed!")
		transactKnownGaps(tx, ctx, 1, slotNum, slotNum, err, "reorg", metrics)
	} else if proposedCount == 0 {
		transactKnownGaps(tx, ctx, 1, slotNum, slotNum, err, "reorg", metrics)
		loghelper.LogReorg(slot, latestBlockRoot).Info("Updated the row that should have been marked as proposed.")
	}

	metrics.IncrementReorgsInsert(1)
}

// Wrapper function that will create a transaction and execute the function.
func writeReorgs(db sql.Database, slot string, latestBlockRoot string, metrics *BeaconClientMetrics) {
	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		loghelper.LogReorgError(slot, latestBlockRoot, err).Fatal("Unable to create a new transaction for reorgs")
	}
	defer func() {
		err := tx.Rollback(ctx)
		if err != nil && err != pgx.ErrTxClosed {
			loghelper.LogError(err).Error("We were unable to Rollback a transaction for reorgs")
		}
	}()
	transactReorgs(tx, ctx, slot, latestBlockRoot, metrics)
	if err = tx.Commit(ctx); err != nil {
		loghelper.LogReorgError(slot, latestBlockRoot, err).Fatal("Unable to execute the transaction for reorgs")
	}
}

// Update the slots table by marking the old slot's as forked.
func updateForked(tx sql.Tx, ctx context.Context, slot string, latestBlockRoot string) (int64, error) {
	res, err := tx.Exec(ctx, UpdateForkedStmt, slot, latestBlockRoot)
	if err != nil {
		loghelper.LogReorgError(slot, latestBlockRoot, err).Error("We are unable to update the ethcl.slots table with the forked slots")
		return 0, err
	}
	count, err := res.RowsAffected()
	if err != nil {
		loghelper.LogReorgError(slot, latestBlockRoot, err).Error("Unable to figure out how many entries were marked as forked.")
		return 0, err
	}
	return count, err
}

// Mark a slot as proposed.
func updateProposed(tx sql.Tx, ctx context.Context, slot string, latestBlockRoot string) (int64, error) {
	res, err := tx.Exec(ctx, UpdateProposedStmt, slot, latestBlockRoot)
	if err != nil {
		loghelper.LogReorgError(slot, latestBlockRoot, err).Error("We are unable to update the ethcl.slots table with the proposed slot.")
		return 0, err
	}
	count, err := res.RowsAffected()
	if err != nil {
		loghelper.LogReorgError(slot, latestBlockRoot, err).Error("Unable to figure out how many entries were marked as proposed")
		return 0, err
	}

	return count, err
}

// A wrapper function to call upsertKnownGaps. This function will break down the range of known_gaps into
// smaller chunks. For example, instead of having an entry of 1-101, if we increment the entries by 10 slots, we would
// have 10 entries as follows: 1-10, 11-20, etc...
func transactKnownGaps(tx sql.Tx, ctx context.Context, tableIncrement int, startSlot int, endSlot int, entryError error, entryProcess string, metric *BeaconClientMetrics) {
	if endSlot-startSlot <= tableIncrement {
		kgModel := DbKnownGaps{
			StartSlot:         strconv.Itoa(startSlot),
			EndSlot:           strconv.Itoa(endSlot),
			CheckedOut:        false,
			ReprocessingError: "",
			EntryError:        entryError.Error(),
			EntryProcess:      entryProcess,
		}
		upsertKnownGaps(tx, ctx, kgModel, metric)
	} else {
		totalSlots := endSlot - startSlot
		var chunks int
		chunks = totalSlots / tableIncrement
		if totalSlots%tableIncrement != 0 {
			chunks = chunks + 1
		}

		for i := 0; i < chunks; i++ {
			var tempStart, tempEnd int
			tempStart = startSlot + (i * tableIncrement)
			if i+1 == chunks {
				tempEnd = endSlot
			} else {
				tempEnd = startSlot + ((i + 1) * tableIncrement)
			}
			kgModel := DbKnownGaps{
				StartSlot:         strconv.Itoa(tempStart),
				EndSlot:           strconv.Itoa(tempEnd),
				CheckedOut:        false,
				ReprocessingError: "",
				EntryError:        entryError.Error(),
				EntryProcess:      entryProcess,
			}
			upsertKnownGaps(tx, ctx, kgModel, metric)
		}
	}
}

// Wrapper function, instead of adding the knownGaps entries to a transaction, it will
// create the transaction and write it.
func writeKnownGaps(db sql.Database, tableIncrement int, startSlot int, endSlot int, entryError error, entryProcess string, metric *BeaconClientMetrics) {
	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		loghelper.LogSlotRangeError(strconv.Itoa(startSlot), strconv.Itoa(endSlot), err).Fatal("Unable to create a new transaction for knownGaps")
	}
	defer func() {
		err := tx.Rollback(ctx)
		if err != nil && err != pgx.ErrTxClosed {
			loghelper.LogError(err).Error("We were unable to Rollback a transaction for reorgs")
		}
	}()
	transactKnownGaps(tx, ctx, tableIncrement, startSlot, endSlot, entryError, entryProcess, metric)
	if err = tx.Commit(ctx); err != nil {
		loghelper.LogSlotRangeError(strconv.Itoa(startSlot), strconv.Itoa(endSlot), err).Fatal("Unable to execute the transaction for knownGaps")
	}
}

// A function to upsert a single entry to the ethcl.known_gaps table.
func upsertKnownGaps(tx sql.Tx, ctx context.Context, knModel DbKnownGaps, metric *BeaconClientMetrics) {
	_, err := tx.Exec(ctx, UpsertKnownGapsStmt, knModel.StartSlot, knModel.EndSlot,
		knModel.CheckedOut, knModel.ReprocessingError, knModel.EntryError, knModel.EntryProcess)
	if err != nil {
		log.WithFields(log.Fields{
			"err":       err,
			"startSlot": knModel.StartSlot,
			"endSlot":   knModel.EndSlot,
		}).Fatal("We are unable to write to the ethcl.known_gaps table!!! We will stop the application because of that.")
	}
	log.WithFields(log.Fields{
		"startSlot": knModel.StartSlot,
		"endSlot":   knModel.EndSlot,
	}).Warn("A new gap has been added to the ethcl.known_gaps table.")
	metric.IncrementKnownGapsInserts(1)
}

// A function to write the gap between the highest slot in the DB and the first processed slot.
func writeStartUpGaps(db sql.Database, tableIncrement int, firstSlot int, metric *BeaconClientMetrics) {
	var maxSlot int
	err := db.QueryRow(context.Background(), QueryHighestSlotStmt).Scan(&maxSlot)
	if err != nil {
		loghelper.LogError(err).Fatal("Unable to get the max block from the DB. We must close the application or we might have undetected gaps.")
	}

	if err != nil {
		loghelper.LogError(err).WithFields(log.Fields{
			"maxSlot": maxSlot,
		}).Fatal("Unable to get convert max block from DB to int. We must close the application or we might have undetected gaps.")
	}
	if maxSlot != firstSlot-1 {
		writeKnownGaps(db, tableIncrement, maxSlot+1, firstSlot-1, fmt.Errorf(""), "startup", metric)
	}
}

// A function to update a knownGap range with a reprocessing error.
func updateKnownGapErrors(db sql.Database, startSlot int, endSlot int, reprocessingErr error, metric *BeaconClientMetrics) error {
	res, err := db.Exec(context.Background(), UpsertKnownGapsErrorStmt, startSlot, endSlot, reprocessingErr.Error())
	if err != nil {
		loghelper.LogSlotRangeError(strconv.Itoa(startSlot), strconv.Itoa(endSlot), err).Error("Unable to update reprocessing_error")
		metric.IncrementKnownGapsProcessingError(1)
		return err
	}
	row, err := res.RowsAffected()
	if err != nil {
		loghelper.LogSlotRangeError(strconv.Itoa(startSlot), strconv.Itoa(endSlot), err).Error("Unable to count rows affected when trying to update reprocessing_error.")
		metric.IncrementKnownGapsProcessingError(1)
		return err
	}
	if row != 1 {
		loghelper.LogSlotRangeError(strconv.Itoa(startSlot), strconv.Itoa(endSlot), err).WithFields(log.Fields{
			"rowCount": row,
		}).Error("The rows affected by the upsert for reprocessing_error is not 1.")
		metric.IncrementKnownGapsProcessingError(1)
		return err
	}
	metric.IncrementKnownGapsProcessed(1)
	return nil
}

// A quick helper function to calculate the epoch.
func calculateEpoch(slot int, slotPerEpoch int) string {
	epoch := slot / slotPerEpoch
	return strconv.Itoa(epoch)
}

// A helper function to check to see if the slot is processed.
func isSlotProcessed(db sql.Database, checkProcessStmt string, slot string) (bool, error) {
	processRow, err := db.Exec(context.Background(), checkProcessStmt, slot)
	if err != nil {
		return false, err
	}
	row, err := processRow.RowsAffected()
	if err != nil {
		return false, err
	}
	if row > 0 {
		return true, nil
	}
	return false, nil
}

// Check to see if this slot is in the DB. Check ethcl.slots, ethcl.signed_beacon_block
// and ethcl.beacon_state. If the slot exists, return true
func IsSlotInDb(db sql.Database, slot string, blockRoot string, stateRoot string) (bool, error) {
	var (
		isInBeaconState       bool
		isInSignedBeaconBlock bool
		err                   error
	)
	errG, _ := errgroup.WithContext(context.Background())
	errG.Go(func() error {
		isInBeaconState, err = checkSlotAndRoot(db, CheckBeaconStateStmt, slot, stateRoot)
		if err != nil {
			loghelper.LogError(err).Error("Unable to check if the slot and stateroot exist in ethcl.beacon_state")
		}
		return err
	})
	errG.Go(func() error {
		isInSignedBeaconBlock, err = checkSlotAndRoot(db, CheckSignedBeaconBlockStmt, slot, blockRoot)
		if err != nil {
			loghelper.LogError(err).Error("Unable to check if the slot and block_root exist in ethcl.signed_beacon_block")
		}
		return err
	})
	if err := errG.Wait(); err != nil {
		return false, err
	}
	if isInBeaconState && isInSignedBeaconBlock {
		return true, nil
	}
	return false, nil
}

// Provide a statement, slot, and root, and this function will check to see
// if the slot and root exist in the table.
func checkSlotAndRoot(db sql.Database, statement, slot, root string) (bool, error) {
	processRow, err := db.Exec(context.Background(), statement, slot, root)
	if err != nil {
		return false, err
	}
	row, err := processRow.RowsAffected()
	if err != nil {
		return false, err
	}
	if row > 0 {
		return true, nil
	}
	return false, nil
}
