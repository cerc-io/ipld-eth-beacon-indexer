package beaconclient

import log "github.com/sirupsen/logrus"

// This function will perform the necessary steps to handle a reorg.
func (bc *BeaconClient) handleReorgs() {
	log.Info("Starting to process reorgs.")
	for {
		// We will add real functionality later
		reorg := <-bc.ReOrgTracking.ProcessCh
		log.WithFields(log.Fields{"reorg": reorg}).Debug("Received a new reorg message.")
	}
}

// This function will perform the necessary steps to handle a reorg.
func (bc *BeaconClient) handleFinalizedCheckpoint() {
	log.Info("Starting to process finalized checkpoints.")
	for {
		// We will add real functionality later
		finalized := <-bc.ReOrgTracking.ProcessCh
		log.WithFields(log.Fields{"finalized": finalized}).Debug("Received a new finalized checkpoint.")
	}

}

// This function will handle the latest head event.
func (bc *BeaconClient) handleHead() {
	log.Info("Starting to process head.")
	for {
		// We will add real functionality later
		head := <-bc.ReOrgTracking.ProcessCh
		log.WithFields(log.Fields{"head": head}).Debug("Received a new head event.")
	}

}
