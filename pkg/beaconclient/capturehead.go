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
func (bc *BeaconClient) CaptureHead(ctx context.Context, maxHeadWorkers int, skipSee bool) {
	log.Info("We are tracking the head of the chain.")
	go bc.handleHead(ctx, maxHeadWorkers)
	go bc.handleReorg(ctx)
	bc.captureEventTopic(ctx, skipSee)
}

// Stop the head tracking service.
func (bc *BeaconClient) StopHeadTracking(ctx context.Context, skipSee bool) {
	select {
	case <-ctx.Done():
		if !skipSee {
			bc.HeadTracking.SseClient.Unsubscribe(bc.HeadTracking.MessagesCh)
			bc.ReOrgTracking.SseClient.Unsubscribe(bc.ReOrgTracking.MessagesCh)
			log.Info("Successfully unsubscribed to SSE client")
			close(bc.ReOrgTracking.MessagesCh)
			close(bc.HeadTracking.MessagesCh)
		}
		log.Info("Successfully stopped the head tracking service.")
	default:
		log.Error("The context has not completed....")
	}
}
