package shutdown

import (
	"context"
	"time"

	"github.com/vulcanize/ipld-ethcl-indexer/pkg/beaconclient"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/gracefulshutdown"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

// Shutdown all the internal services for the application.
func ShutdownServices(ctx context.Context, waitTime time.Duration, DB sql.Database, BC *beaconclient.BeaconClient) <-chan struct{} {
	return gracefulshutdown.Shutdown(ctx, waitTime, map[string]gracefulshutdown.Operation{
		"database": func(ctx context.Context) error {
			err := DB.Close()
			if err != nil {
				loghelper.LogError(err).Error("Unable to close the DB")
			}
			return err
		},
		"beaconClient": func(ctx context.Context) error {
			err := BC.StopHeadTracking()
			if err != nil {
				loghelper.LogError(err).Error("Unable to trigger shutdown of head tracking")
			}
			return err
		},
	})
}
