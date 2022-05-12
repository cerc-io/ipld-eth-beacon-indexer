package beaconclient

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

// The sync response
type Sync struct {
	Data SyncData `json:"data"`
}

// The sync data
type SyncData struct {
	IsSync       bool   `json:"is_syncing"`
	HeadSlot     string `json:"head_slot"`
	SyncDistance string `json:"sync_distance"`
}

// This function will check to see if we are synced up with the head of chain.
//{"data":{"is_syncing":true,"head_slot":"62528","sync_distance":"3734299"}}
func (bc BeaconClient) CheckHeadSync() (bool, error) {
	bcSync := bc.ServerEndpoint + BcSyncStatusEndpoint
	resp, err := http.Get(bcSync)

	if err != nil {
		loghelper.LogEndpoint(bcSync).Error("Unable to check the sync status")
		return true, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		loghelper.LogEndpoint(bcSync).WithFields(log.Fields{"returnCode": resp.StatusCode}).Error("Error when getting the sync status")
		return true, fmt.Errorf("Querying the sync status returned a non 2xx status code, code provided: %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return true, err
	}

	var syncStatus Sync
	if err := json.Unmarshal(body, &syncStatus); err != nil {
		loghelper.LogEndpoint(bcSync).WithFields(log.Fields{
			"rawMessage": string(body),
			"err":        err,
		}).Error("Unable to unmarshal sync status")
		return true, err
	}

	return syncStatus.Data.IsSync, nil
}
