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
