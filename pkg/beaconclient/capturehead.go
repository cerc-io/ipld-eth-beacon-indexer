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
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

// This function will perform all the heavy lifting for tracking the head of the chain.
func (bc *BeaconClient) CaptureHead() {
	log.Info("We are tracking the head of the chain.")
	go bc.handleHead()
	go bc.handleReorg()
	bc.captureEventTopic()
}

// Stop the head tracking service.
func (bc *BeaconClient) StopHeadTracking() error {
	log.Info("We are going to stop tracking the head of chain because of the shutdown signal.")
	chHead := make(chan bool)
	chReorg := make(chan bool)

	go bc.HeadTracking.finishProcessingChannel(chHead)
	go bc.ReOrgTracking.finishProcessingChannel(chReorg)

	<-chHead
	<-chReorg
	log.Info("Successfully stopped the head tracking service.")
	return nil
}

// This function closes the SSE subscription, but waits until the MessagesCh is empty
func (se *SseEvents[ProcessedEvents]) finishProcessingChannel(finish chan<- bool) {
	loghelper.LogEndpoint(se.Endpoint).Info("Received a close event.")
	se.SseClient.Unsubscribe(se.MessagesCh)
	for len(se.MessagesCh) != 0 || len(se.ProcessCh) != 0 {
		time.Sleep(time.Duration(shutdownWaitInterval) * time.Millisecond)
	}
	loghelper.LogEndpoint(se.Endpoint).Info("Done processing all messages, ready for shutdown")
	finish <- true
}
