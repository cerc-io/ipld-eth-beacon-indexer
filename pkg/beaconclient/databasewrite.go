package beaconclient

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

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
INSERT INTO ethcl.signed_beacon_block (slot, block_root, parent_block_root, mh_key)
VALUES ($1, $2, $3, $4) ON CONFLICT (slot, block_root) DO NOTHING`
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

func CreateDatabaseWrite(db sql.Database, slot int, stateRoot string, blockRoot string, parentBlockRoot string, status string, metrics *BeaconClientMetrics) *DatabaseWriter {
	dw := &DatabaseWriter{
		Db:      db,
		Metrics: metrics,
	}
	dw.prepareSlotsModel(slot, stateRoot, blockRoot, status)
	dw.prepareSignedBeaconBlockModel(slot, blockRoot, parentBlockRoot)
	dw.prepareBeaconStateModel(slot, stateRoot)
	return dw
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
func (dw *DatabaseWriter) prepareSignedBeaconBlockModel(slot int, blockRoot string, parentBlockRoot string) {
	dw.DbSignedBeaconBlock = &DbSignedBeaconBlock{
		Slot:        strconv.Itoa(slot),
		BlockRoot:   blockRoot,
		ParentBlock: parentBlockRoot,
		MhKey:       calculateMhKey(),
	}
	log.Debug("dw.DbSignedBeaconBlock: ", dw.DbSignedBeaconBlock)
}

// Create the model for the ethcl.beacon_state table.
func (dw *DatabaseWriter) prepareBeaconStateModel(slot int, stateRoot string) {
	dw.DbBeaconState = &DbBeaconState{
		Slot:      strconv.Itoa(slot),
		StateRoot: stateRoot,
		MhKey:     calculateMhKey(),
	}

	log.Debug("dw.DbBeaconState: ", dw.DbBeaconState)
}

// Write all the data for a given slot.
func (dw *DatabaseWriter) writeFullSlot() {
	// Add errors for each function call
	// If an error occurs, write to knownGaps table.
	dw.writeSlots()
	dw.writeSignedBeaconBlocks()
	dw.writeBeaconState()
	dw.Metrics.IncrementHeadTrackingInserts(1)
}

// Write the information for the generic slots table. For now this is only one function.
// But in the future if we need to incorporate any FK's or perform any actions to write to the
// slots table we can do it all here.
func (dw *DatabaseWriter) writeSlots() {
	dw.upsertSlots()

}

// Upsert to the ethcl.slots table.
func (dw *DatabaseWriter) upsertSlots() {
	_, err := dw.Db.Exec(context.Background(), UpsertSlotsStmt, dw.DbSlots.Epoch, dw.DbSlots.Slot, dw.DbSlots.BlockRoot, dw.DbSlots.StateRoot, dw.DbSlots.Status)
	if err != nil {
		loghelper.LogSlotError(dw.DbSlots.Slot, err).Error("Unable to write to the slot to the ethcl.slots table")
	}
}

// Write the information for the signed_beacon_block.
func (dw *DatabaseWriter) writeSignedBeaconBlocks() {
	dw.upsertPublicBlocks(dw.DbSignedBeaconBlock.MhKey, dw.rawSignedBeaconBlock)
	dw.upsertSignedBeaconBlock()
}

// Upsert to public.blocks.
func (dw *DatabaseWriter) upsertPublicBlocks(key string, data []byte) {
	_, err := dw.Db.Exec(context.Background(), UpsertBlocksStmt, key, data)
	if err != nil {
		loghelper.LogSlotError(dw.DbSlots.Slot, err).Error("Unable to write to the slot to the public.blocks table")
	}
}

// Upsert to the ethcl.signed_beacon_block table.
func (dw *DatabaseWriter) upsertSignedBeaconBlock() {
	_, err := dw.Db.Exec(context.Background(), UpsertSignedBeaconBlockStmt, dw.DbSignedBeaconBlock.Slot, dw.DbSignedBeaconBlock.BlockRoot, dw.DbSignedBeaconBlock.ParentBlock, dw.DbSignedBeaconBlock.MhKey)
	if err != nil {
		loghelper.LogSlotError(dw.DbSlots.Slot, err).WithFields(log.Fields{"block_root": dw.DbSignedBeaconBlock.BlockRoot}).Error("Unable to write to the slot to the ethcl.signed_beacon_block table")
	}
}

// Write the information for the beacon_state.
func (dw *DatabaseWriter) writeBeaconState() {
	dw.upsertPublicBlocks(dw.DbBeaconState.MhKey, dw.rawBeaconState)
	dw.upsertBeaconState()
}

// Upsert to the ethcl.beacon_state table.
func (dw *DatabaseWriter) upsertBeaconState() {
	_, err := dw.Db.Exec(context.Background(), UpsertBeaconState, dw.DbBeaconState.Slot, dw.DbBeaconState.StateRoot, dw.DbBeaconState.MhKey)
	if err != nil {
		loghelper.LogSlotError(dw.DbSlots.Slot, err).Error("Unable to write to the slot to the ethcl.beacon_state table")
	}
}

// Update a given slot to be marked as forked. Provide the slot and the latest latestBlockRoot.
// We will mark all entries for the given slot that don't match the provided latestBlockRoot as forked.
func writeReorgs(db sql.Database, slot string, latestBlockRoot string, metrics *BeaconClientMetrics) {
	forkCount, err := updateForked(db, slot, latestBlockRoot)
	if err != nil {
		loghelper.LogReorgError(slot, latestBlockRoot, err).Error("We ran into some trouble while updating all forks.")
		// Add to knownGaps Table
	}
	proposedCount, err := updateProposed(db, slot, latestBlockRoot)
	if err != nil {
		loghelper.LogReorgError(slot, latestBlockRoot, err).Error("We ran into some trouble while trying to update the proposed slot.")
		// Add to knownGaps Table
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
	} else if proposedCount == 0 {
		var count int
		err := db.QueryRow(context.Background(), CheckProposedStmt, slot, latestBlockRoot).Scan(count)
		if err != nil {
			loghelper.LogReorgError(slot, latestBlockRoot, err).Error("Unable to query proposed rows after reorg.")
		}
		if count != 1 {
			loghelper.LogReorg(slot, latestBlockRoot).WithFields(log.Fields{
				"proposedCount": count,
			}).Warn("The proposed block was not marked as proposed...")
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

// Dummy function for calculating the mhKey.
func calculateMhKey() string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 10)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[:10]
}

// A quick helper function to calculate the epoch.
func calculateEpoch(slot int, slotPerEpoch int) string {
	epoch := slot / slotPerEpoch
	return strconv.Itoa(epoch)
}
