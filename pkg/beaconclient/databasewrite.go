package beaconclient

import "github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"

type DatabaseWriter struct {
	Db                  sql.Database
	DbSlots             *DbSlots
	DbSignedBeaconBlock *DbSignedBeaconBlock
	DbBeaconState       *DbBeaconState
}

// Write functions to write each all together...
// Should I do one atomic write?
