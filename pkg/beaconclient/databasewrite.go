package beaconclient

import (
	"context"
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
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
	// Statement to insert known_gaps. We don't pass in timestamp, we let the server take care of that one.
	UpsertKnownGapsStmt string = `
INSERT INTO ethcl.known_gaps (start_slot, end_slot, checked_out, reprocessing_error, entry_error, entry_process)
VALUES ($1, $2, $3, $4, $5, $6) on CONFLICT (start_slot, end_slot) DO NOTHING`
	QueryHighestSlotStmt string = "SELECT COALESCE(MAX(slot), 0) FROM ethcl.slots"
)

// Put all functionality to prepare the write object
// And write it in this file.
// Remove any of it from the processslot file.
type DatabaseWriter struct {
	Db                   sql.Database
	Metrics              *BeaconClientMetrics
	DbSlots              *DbSlots
	DbSignedBeaconBlock  *DbSignedBeaconBlock
	DbBeaconState        *DbBeaconState
	rawBeaconState       []byte
	rawSignedBeaconBlock []byte
}

func CreateDatabaseWrite(db sql.Database, slot int, stateRoot string, blockRoot string, parentBlockRoot string,
	eth1BlockHash string, status string, rawSignedBeaconBlock []byte, rawBeaconState []byte, metrics *BeaconClientMetrics) (*DatabaseWriter, error) {
	dw := &DatabaseWriter{
		Db:                   db,
		rawBeaconState:       rawBeaconState,
		rawSignedBeaconBlock: rawSignedBeaconBlock,
		Metrics:              metrics,
	}
	dw.prepareSlotsModel(slot, stateRoot, blockRoot, status)
	err := dw.prepareSignedBeaconBlockModel(slot, blockRoot, parentBlockRoot, eth1BlockHash)
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

// Write all the data for a given slot.
func (dw *DatabaseWriter) writeFullSlot() error {
	// If an error occurs, write to knownGaps table.
	log.WithFields(log.Fields{
		"slot": dw.DbSlots.Slot,
	}).Debug("Starting to write to the DB.")
	err := dw.writeSlots()
	if err != nil {
		return err
	}
	if dw.DbSlots.Status != "skipped" {
		err = dw.writeSignedBeaconBlocks()
		if err != nil {
			return err
		}
		err = dw.writeBeaconState()
		if err != nil {
			return err
		}
	}
	dw.Metrics.IncrementHeadTrackingInserts(1)
	return nil
}

// Write the information for the generic slots table. For now this is only one function.
// But in the future if we need to incorporate any FK's or perform any actions to write to the
// slots table we can do it all here.
func (dw *DatabaseWriter) writeSlots() error {
	return dw.upsertSlots()
}

// Upsert to the ethcl.slots table.
func (dw *DatabaseWriter) upsertSlots() error {
	_, err := dw.Db.Exec(context.Background(), UpsertSlotsStmt, dw.DbSlots.Epoch, dw.DbSlots.Slot, dw.DbSlots.BlockRoot, dw.DbSlots.StateRoot, dw.DbSlots.Status)
	if err != nil {
		loghelper.LogSlotError(dw.DbSlots.Slot, err).Error("Unable to write to the slot to the ethcl.slots table")
		return err
	}
	return nil
}

// Write the information for the signed_beacon_block.
func (dw *DatabaseWriter) writeSignedBeaconBlocks() error {
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
	_, err := dw.Db.Exec(context.Background(), UpsertBlocksStmt, key, data)
	if err != nil {
		loghelper.LogSlotError(dw.DbSlots.Slot, err).Error("Unable to write to the slot to the public.blocks table")
		return err
	}
	return nil
}

// Upsert to the ethcl.signed_beacon_block table.
func (dw *DatabaseWriter) upsertSignedBeaconBlock() error {
	_, err := dw.Db.Exec(context.Background(), UpsertSignedBeaconBlockStmt, dw.DbSignedBeaconBlock.Slot, dw.DbSignedBeaconBlock.BlockRoot, dw.DbSignedBeaconBlock.ParentBlock, dw.DbSignedBeaconBlock.Eth1BlockHash, dw.DbSignedBeaconBlock.MhKey)
	if err != nil {
		loghelper.LogSlotError(dw.DbSlots.Slot, err).WithFields(log.Fields{"block_root": dw.DbSignedBeaconBlock.BlockRoot}).Error("Unable to write to the slot to the ethcl.signed_beacon_block table")
		return err
	}
	return nil
}

// Write the information for the beacon_state.
func (dw *DatabaseWriter) writeBeaconState() error {
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
	_, err := dw.Db.Exec(context.Background(), UpsertBeaconState, dw.DbBeaconState.Slot, dw.DbBeaconState.StateRoot, dw.DbBeaconState.MhKey)
	if err != nil {
		loghelper.LogSlotError(dw.DbSlots.Slot, err).Error("Unable to write to the slot to the ethcl.beacon_state table")
		return err
	}
	return nil
}

// Update a given slot to be marked as forked. Provide the slot and the latest latestBlockRoot.
// We will mark all entries for the given slot that don't match the provided latestBlockRoot as forked.
func writeReorgs(db sql.Database, slot string, latestBlockRoot string, metrics *BeaconClientMetrics) {
	slotNum, strErr := strconv.Atoi(slot)
	if strErr != nil {
		loghelper.LogReorgError(slot, latestBlockRoot, strErr).Error("We can't convert the slot to an int...")
	}

	forkCount, err := updateForked(db, slot, latestBlockRoot)
	if err != nil {
		loghelper.LogReorgError(slot, latestBlockRoot, err).Error("We ran into some trouble while updating all forks.")
		writeKnownGaps(db, 1, slotNum, slotNum, err, "reorg", metrics)
	}
	proposedCount, err := updateProposed(db, slot, latestBlockRoot)
	if err != nil {
		loghelper.LogReorgError(slot, latestBlockRoot, err).Error("We ran into some trouble while trying to update the proposed slot.")
		writeKnownGaps(db, 1, slotNum, slotNum, err, "reorg", metrics)
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
		writeKnownGaps(db, 1, slotNum, slotNum, err, "reorg", metrics)
	} else if proposedCount == 0 {
		var count int
		err := db.QueryRow(context.Background(), CheckProposedStmt, slot, latestBlockRoot).Scan(count)
		if err != nil {
			loghelper.LogReorgError(slot, latestBlockRoot, err).Error("Unable to query proposed rows after reorg.")
			writeKnownGaps(db, 1, slotNum, slotNum, err, "reorg", metrics)
		}
		if count != 1 {
			loghelper.LogReorg(slot, latestBlockRoot).WithFields(log.Fields{
				"proposedCount": count,
			}).Warn("The proposed block was not marked as proposed...")
			writeKnownGaps(db, 1, slotNum, slotNum, err, "reorg", metrics)
		} else {
			loghelper.LogReorg(slot, latestBlockRoot).Info("Updated the row that should have been marked as proposed.")
		}
	}

	metrics.IncrementHeadTrackingReorgs(1)
}

// Update the slots table by marking the old slot's as forked.
func updateForked(db sql.Database, slot string, latestBlockRoot string) (int64, error) {
	res, err := db.Exec(context.Background(), UpdateForkedStmt, slot, latestBlockRoot)
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

func updateProposed(db sql.Database, slot string, latestBlockRoot string) (int64, error) {
	res, err := db.Exec(context.Background(), UpdateProposedStmt, slot, latestBlockRoot)
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

// A wrapper function to call upsertKnownGaps. This function will break down the range of known_gaos into
// smaller chunks. For example, instead of having an entry of 1-101, if we increment the entries by 10 slots, we would
// have 10 entries as follows: 1-10, 11-20, etc...
func writeKnownGaps(db sql.Database, tableIncrement int, startSlot int, endSlot int, entryError error, entryProcess string, metric *BeaconClientMetrics) {
	if endSlot-startSlot <= tableIncrement {
		kgModel := DbKnownGaps{
			StartSlot:         strconv.Itoa(startSlot),
			EndSlot:           strconv.Itoa(endSlot),
			CheckedOut:        false,
			ReprocessingError: "",
			EntryError:        entryError.Error(),
			EntryProcess:      entryProcess,
		}
		upsertKnownGaps(db, kgModel)
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
			upsertKnownGaps(db, kgModel)
		}
	}
	metric.IncrementHeadTrackingKnownGaps(1)

}

// A function to upsert a single entry to the ethcl.known_gaps table.
func upsertKnownGaps(db sql.Database, knModel DbKnownGaps) {
	_, err := db.Exec(context.Background(), UpsertKnownGapsStmt, knModel.StartSlot, knModel.EndSlot,
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
		writeKnownGaps(db, tableIncrement, maxSlot, firstSlot-1, fmt.Errorf(""), "startup", metric)
	}
}

// A quick helper function to calculate the epoch.
func calculateEpoch(slot int, slotPerEpoch int) string {
	epoch := slot / slotPerEpoch
	return strconv.Itoa(epoch)
}
