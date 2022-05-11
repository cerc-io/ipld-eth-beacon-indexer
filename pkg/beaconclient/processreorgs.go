package beaconclient

import (
	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

func processReorg(db sql.Database, slot string, latestBlockRoot string, metrics *BeaconClientMetrics) {
	updatedRows, err := updateReorgs(db, slot, latestBlockRoot, metrics)

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
