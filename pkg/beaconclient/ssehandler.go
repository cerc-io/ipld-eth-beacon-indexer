// This package will handle all event subscriptions that utilize SSE.

package beaconclient

import (
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

var (
	shutdownWaitInterval = time.Duration(5) * time.Second
)

// This function will capture all the SSE events for a given SseEvents object.
// When new messages come in, it will ensure that they are decoded into JSON.
// If any errors occur, it log the error information.
func handleSseEvent[P ProcessedEvents](eventHandler *SseEvents[P]) {
	loghelper.LogUrl(eventHandler.Url).Info("Subscribing to Messages")
	go eventHandler.SseClient.SubscribeChanRaw(eventHandler.MessagesCh)
	for {
		select {
		case message := <-eventHandler.MessagesCh:
			// Message can be nil if its a keep-alive message
			if len(message.Data) != 0 {
				go processMsg(message.Data, eventHandler.ProcessCh, eventHandler.ErrorCh)
			}

		case headErr := <-eventHandler.ErrorCh:
			log.WithFields(log.Fields{
				"url": eventHandler.Url,
				"err": headErr.err,
				"msg": headErr.msg,
			},
			).Error("Unable to handle event.")

		case process := <-eventHandler.ProcessCh:
			log.WithFields(log.Fields{"processed": process}).Debug("Processesing a Message")
		}
	}
}

// Capture all of the head topics.
func (bc *BeaconClient) CaptureHeadTopic() {
	log.Info("We are capturing all SSE events")
	go handleSseEvent(bc.HeadTracking)
	go handleSseEvent(bc.ReOrgTracking)
	go handleSseEvent(bc.FinalizationTracking)
}

// Turn the data object into a Struct.
func processMsg[P ProcessedEvents](msg []byte, processCh chan<- *P, errorCh chan<- *SseError) {
	log.WithFields(log.Fields{"msg": msg}).Debug("Processing a Message")
	var msgMarshaled P
	err := json.Unmarshal(msg, &msgMarshaled)
	if err != nil {
		errorCh <- &SseError{
			err: err,
			msg: msg,
		}
		return
	}
	processCh <- &msgMarshaled
}
