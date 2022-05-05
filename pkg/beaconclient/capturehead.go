// This file will call all the functions to start and stop capturing the head of the beacon chain.

package beaconclient

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

// This function will perform all the heavy lifting for tracking the head of the chain.
func (bc *BeaconClient) CaptureHead(db sql.Database) {
	log.Info("We are tracking the head of the chain.")
	//bc.tempHelper()
	go bc.handleHead(db)
	//go bc.handleFinalizedCheckpoint()
	go bc.handleReorgs()
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
