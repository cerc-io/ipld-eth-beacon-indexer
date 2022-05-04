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
func ShutdownServices(ctx context.Context, waitTime time.Duration, DB sql.Database, BC *beaconclient.BeaconClient) error {
	successCh, errCh := gracefulshutdown.Shutdown(ctx, waitTime, map[string]gracefulshutdown.Operation{
		// Combining DB shutdown with BC because BC needs DB open to cleanly shutdown.
		"beaconClient": func(ctx context.Context) error {
			defer DB.Close()
			err := BC.StopHeadTracking()
			if err != nil {
				loghelper.LogError(err).Error("Unable to trigger shutdown of head tracking")
			}
			return err
		},
	})

	select {
	case _ = <-successCh:
		return nil
	case err := <-errCh:
		return err
	}
}
