package beaconclient

import (
	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

func processReorg(db sql.Database, slot string, latestBlockRoot string) {
	// Check to see if there are slots in the DB with the given slot.
	// Update them ALL to forked
	// Upsert the new slot into the DB, mark the status to proposed.
	// Query at the end to make sure that you have handled the reorg properly.
	updatedRows, err := updateReorgs(db, slot, latestBlockRoot)

	if err != nil {
		// Add this slot to the knownGaps table..
		// Maybe we need to rename the knownGaps table to the "batchProcess" table.
	}

	if updatedRows > 0 {
		loghelper.LogReorg(slot, latestBlockRoot).WithFields(log.Fields{
			"updatedRows": updatedRows,
		}).Info("Updated DB based on Reorgs.")
	} else {
		loghelper.LogReorg(slot, latestBlockRoot).WithFields(log.Fields{
			"updatedRows": updatedRows,
		}).Warn("There were no rows to update.")

	}
}
