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

// A temporary helper function to see the output of beacon block and states.
//func (bc *BeaconClient) tempHelper() {
//	slot := "3200"
//	blockEndpoint := bc.ServerEndpoint + bcBlockQueryEndpoint + slot
//	stateEndpoint := bc.ServerEndpoint + bcStateQueryEndpoint + slot
//	// Query
//	log.Info("Get")
//	blockSsz, _ := querySsz(blockEndpoint, slot)
//	stateSsz, _ := querySsz(stateEndpoint, slot)
//	// Transform
//	log.Info("Tranform")
//	stateObj := new(spectests.BeaconState)
//	err := stateObj.UnmarshalSSZ(stateSsz)
//	if err != nil {
//		loghelper.LogSlotError(slot, err).Error("Unable to unmarshal the SSZ response from the Beacon Node Successfully!")
//	}
//
//	blockObj := new(spectests.SignedBeaconBlock)
//	err = blockObj.UnmarshalSSZ(blockSsz)
//	if err != nil {
//		loghelper.LogSlotError(slot, err).Error("Unable to unmarshal the SSZ response from the Beacon Node Successfully!")
//	}
//
//	// Check
//	log.Info("Check")
//	log.Info("State Slot: ", stateObj.Slot)
//	log.Info("Block Slot: ", blockObj.Block.Slot)
//}
//
// Stop the head tracking service.
func (bc *BeaconClient) StopHeadTracking() error {
	log.Info("We are going to stop tracking the head of chain because of the shutdown signal.")
	chHead := make(chan bool)
	chReorg := make(chan bool)
	//chFinal := make(chan bool)

	go bc.HeadTracking.finishProcessingChannel(chHead)
	go bc.ReOrgTracking.finishProcessingChannel(chReorg)
	//go bc.FinalizationTracking.finishProcessingChannel(chFinal)

	<-chHead
	//<-chFinal
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
