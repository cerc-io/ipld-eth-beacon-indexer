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
// This package will handle all event subscriptions that utilize SSE.

package beaconclient

import (
	"context"
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/loghelper"
)

// This function will capture all the SSE events for a given SseEvents object.
// When new messages come in, it will ensure that they are decoded into JSON.
// If any errors occur, it log the error information.
func handleIncomingSseEvent[P ProcessedEvents](ctx context.Context, eventHandler *SseEvents[P], errMetricInc func(uint64), skipSse bool) {
	if !skipSse {
		for {
			err := eventHandler.SseClient.SubscribeChanRawWithContext(ctx, eventHandler.MessagesCh)
			if err != nil {
				loghelper.LogEndpoint(eventHandler.Endpoint).WithFields(log.Fields{
					"err": err}).Error("We are unable to subscribe to the SSE endpoint")
				time.Sleep(3 * time.Second)
				continue
			}
			loghelper.LogEndpoint(eventHandler.Endpoint).Info("Successfully subscribed to the event stream.")
			break
		}
	}

	for {
		select {
		case <-ctx.Done():
			close(eventHandler.MessagesCh)
			close(eventHandler.ErrorCh)
			return
		case message := <-eventHandler.MessagesCh:
			// Message can be nil if its a keep-alive message
			if len(message.Data) != 0 {
				log.WithFields(log.Fields{"msg": string(message.Data)}).Debug("We are going to send the following message to be processed.")
				go processMsg(message.Data, eventHandler.ProcessCh, eventHandler.ErrorCh)
			}

		case headErr := <-eventHandler.ErrorCh:
			log.WithFields(log.Fields{
				"endpoint": eventHandler.Endpoint,
				"err":      headErr.err,
				"msg":      headErr.msg,
			},
			).Error("Unable to handle event.")
			errMetricInc(1)
		}
	}
}

// Turn the data object into a Struct.
func processMsg[P ProcessedEvents](msg []byte, processCh chan<- P, errorCh chan<- SseError) {
	var msgMarshaled P
	err := json.Unmarshal(msg, &msgMarshaled)
	if err != nil {
		loghelper.LogError(err).Error("Unable to parse message")
		errorCh <- SseError{
			err: err,
			msg: msg,
		}
		return
	}
	processCh <- msgMarshaled
	log.Debug("Done sending")
}

// Capture all of the event topics.
func (bc *BeaconClient) captureEventTopic(ctx context.Context, skipSse bool) {
	log.Info("We are capturing all SSE events")
	go handleIncomingSseEvent(ctx, bc.HeadTracking, bc.Metrics.IncrementHeadError, skipSse)
	go handleIncomingSseEvent(ctx, bc.ReOrgTracking, bc.Metrics.IncrementReorgError, skipSse)
}
