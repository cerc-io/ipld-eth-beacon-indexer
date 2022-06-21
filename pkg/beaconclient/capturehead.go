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
// This file will call all the functions to start and stop capturing the head of the beacon chain.

package beaconclient

import (
	"context"

	log "github.com/sirupsen/logrus"
)

// This function will perform all the heavy lifting for tracking the head of the chain.
func (bc *BeaconClient) CaptureHead(ctx context.Context) {
	log.Info("We are tracking the head of the chain.")
	go bc.handleHead(ctx)
	go bc.handleReorg(ctx)
	bc.captureEventTopic(ctx)
}

// Stop the head tracking service.
func (bc *BeaconClient) StopHeadTracking(cancel context.CancelFunc) error {
	log.Info("We are going to stop tracking the head of chain because of the shutdown signal.")
	cancel()
	log.Info("Successfully stopped the head tracking service.")
	return nil
}
