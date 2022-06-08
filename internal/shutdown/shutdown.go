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
package shutdown

import (
	"context"
	"os"
	"time"

	"github.com/vulcanize/ipld-ethcl-indexer/pkg/beaconclient"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/gracefulshutdown"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

// Shutdown all the internal services for the application.
func ShutdownServices(ctx context.Context, notifierCh chan os.Signal, waitTime time.Duration, DB sql.Database, BC *beaconclient.BeaconClient, shutdownOperations map[string]gracefulshutdown.Operation) error {
	//successCh, errCh := gracefulshutdown.Shutdown(ctx, notifierCh, waitTime, )
	successCh, errCh := gracefulshutdown.Shutdown(ctx, notifierCh, waitTime, shutdownOperations)

	select {
	case <-successCh:
		return nil
	case err := <-errCh:
		return err
	}
}

// Wrapper function for shutting down the head tracking process.
func ShutdownHeadTracking(ctx context.Context, kgCancel context.CancelFunc, notifierCh chan os.Signal, waitTime time.Duration, DB sql.Database, BC *beaconclient.BeaconClient) error {
	return ShutdownServices(ctx, notifierCh, waitTime, DB, BC, map[string]gracefulshutdown.Operation{
		// Combining DB shutdown with BC because BC needs DB open to cleanly shutdown.
		"beaconClient": func(ctx context.Context) error {
			defer DB.Close()
			err := BC.StopHeadTracking()
			if err != nil {
				loghelper.LogError(err).Error("Unable to trigger shutdown of head tracking")
			}
			if BC.KnownGapsProcess != (beaconclient.KnownGapsProcessing{}) {
				err = BC.StopKnownGapsProcessing(kgCancel)
				if err != nil {
					loghelper.LogError(err).Error("Unable to stop processing known gaps")
				}
			}
			return err
		},
	})
}

// Wrapper function for shutting down the head tracking process.
func ShutdownHistoricProcessing(ctx context.Context, kgCancel, hpCancel context.CancelFunc, notifierCh chan os.Signal, waitTime time.Duration, DB sql.Database, BC *beaconclient.BeaconClient) error {
	return ShutdownServices(ctx, notifierCh, waitTime, DB, BC, map[string]gracefulshutdown.Operation{
		// Combining DB shutdown with BC because BC needs DB open to cleanly shutdown.
		"beaconClient": func(ctx context.Context) error {
			defer DB.Close()
			err := BC.StopHistoric(hpCancel)
			if err != nil {
				loghelper.LogError(err).Error("Unable to stop processing historic")
			}
			if BC.KnownGapsProcess != (beaconclient.KnownGapsProcessing{}) {
				err = BC.StopKnownGapsProcessing(kgCancel)
				if err != nil {
					loghelper.LogError(err).Error("Unable to stop processing known gaps")
				}
			}
			return err
		},
	})
}

// Wrapper function for shutting down the application in boot mode.
func ShutdownBoot(ctx context.Context, notifierCh chan os.Signal, waitTime time.Duration, DB sql.Database, BC *beaconclient.BeaconClient) error {
	return ShutdownServices(ctx, notifierCh, waitTime, DB, BC, map[string]gracefulshutdown.Operation{
		// Combining DB shutdown with BC because BC needs DB open to cleanly shutdown.
		"Database": func(ctx context.Context) error {
			return DB.Close()
		},
	})
}
