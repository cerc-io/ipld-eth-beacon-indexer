// This package will handle all event subscriptions that utilize SSE.

package beaconclient

import (
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
	"golang.org/x/sync/errgroup"
)

var (
	shutdownWaitInterval = time.Duration(5) * time.Second
)

// This function will capture all the SSE events for a given SseEvents object.
// When new messages come in, it will ensure that they are decoded into JSON.
// If any errors occur, it log the error information.
func handleIncomingSseEvent[P ProcessedEvents](eventHandler *SseEvents[P]) {
	errG := new(errgroup.Group)
	errG.Go(func() error {
		err := eventHandler.SseClient.SubscribeChanRaw(eventHandler.MessagesCh)
		if err != nil {
			return err
		}
		return nil
	})
	if err := errG.Wait(); err != nil {
		log.WithFields(log.Fields{
			"err":      err,
			"endpoint": eventHandler.Endpoint,
		}).Error("Unable to subscribe to the SSE endpoint.")
		return
	} else {
		loghelper.LogEndpoint(eventHandler.Endpoint).Info("Successfully subscribed to the event stream.")
	}
	for {
		select {
		case message := <-eventHandler.MessagesCh:
			// Message can be nil if its a keep-alive message
			if len(message.Data) != 0 {
				go processMsg(message.Data, eventHandler.ProcessCh, eventHandler.ErrorCh)
			}

		case headErr := <-eventHandler.ErrorCh:
			log.WithFields(log.Fields{
				"endpoint": eventHandler.Endpoint,
				"err":      headErr.err,
				"msg":      headErr.msg,
			},
			).Error("Unable to handle event.")

		case process := <-eventHandler.ProcessCh:
			log.WithFields(log.Fields{"processed": process}).Debug("Processesing a Message")
		}
	}
}

// Turn the data object into a Struct.
func processMsg[P ProcessedEvents](msg []byte, processCh chan<- *P, errorCh chan<- *SseError) {
	var msgMarshaled P
	err := json.Unmarshal(msg, &msgMarshaled)
	if err != nil {
		loghelper.LogError(err).Error("Unable to parse message")
		errorCh <- &SseError{
			err: err,
			msg: msg,
		}
		return
	}
	processCh <- &msgMarshaled
}

// Capture all of the event topics.
func (bc *BeaconClient) captureEventTopic() {
	log.Info("We are capturing all SSE events")
	go handleIncomingSseEvent(bc.HeadTracking)
	go handleIncomingSseEvent(bc.ReOrgTracking)
}
