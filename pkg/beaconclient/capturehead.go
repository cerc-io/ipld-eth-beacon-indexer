package beaconclient

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

// This function will perform all the heavy lifting for tracking the head of the chain.
func (bc *BeaconClient) CaptureHead() {
	log.Info("We are tracking the head of the chain.")
	//go readProcessedEvents(bc.HeadTracking.ProcessCh)
	bc.CaptureHeadTopic()
}

// Stop the head tracking service.
func (bc *BeaconClient) StopHeadTracking() error {
	log.Info("We are going to stop tracking the head of chain because of the shutdown signal.")
	chHead := make(chan bool)
	chReorg := make(chan bool)
	chFinal := make(chan bool)

	go bc.HeadTracking.finishProcessingChannel(chHead)
	go bc.ReOrgTracking.finishProcessingChannel(chReorg)
	go bc.FinalizationTracking.finishProcessingChannel(chFinal)

	<-chHead
	<-chFinal
	<-chReorg
	log.Info("Successfully stopped the head tracking service.")
	return nil
}

func (se *SseEvents[ProcessedEvents]) finishProcessingChannel(finish chan<- bool) {
	loghelper.LogUrl(se.Url).Info("Received a close event.")
	se.SseClient.Unsubscribe(se.MessagesCh)
	for len(se.MessagesCh) != 0 {
		time.Sleep(time.Duration(shutdownWaitInterval) * time.Millisecond)
	}
	loghelper.LogUrl(se.Url).Info("Done processing all messages, ready for shutdown")
	finish <- true
}
