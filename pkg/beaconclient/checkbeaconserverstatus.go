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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/loghelper"
)

var (
	MissingBeaconServerType error = fmt.Errorf("The beacon server type provided is not handled.")
	LighthouseMissingSlots  error = fmt.Errorf("Anchor is not nil. This means lighthouse has not backfilled all the slots from Genesis to head.")
)

// The sync response when checking if the node is synced.
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
// {"data":{"is_syncing":true,"head_slot":"62528","sync_distance":"3734299"}}
func (bc BeaconClient) CheckHeadSync() (bool, error) {
	syncStatus, err := bc.QueryHeadSync()
	if err != nil {
		return true, nil
	}
	return syncStatus.Data.IsSync, nil
}

func (bc BeaconClient) QueryHeadSync() (Sync, error) {
	var syncStatus Sync
	bcSync := bc.ServerEndpoint + BcSyncStatusEndpoint
	resp, err := http.Get(bcSync)

	if err != nil {
		loghelper.LogEndpoint(bcSync).Error("Unable to check the sync status")
		return syncStatus, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		loghelper.LogEndpoint(bcSync).WithFields(log.Fields{"returnCode": resp.StatusCode}).Error("Error when getting the sync status")
		return syncStatus, fmt.Errorf("Querying the sync status returned a non 2xx status code, code provided: %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	var body bytes.Buffer
	buf := bufio.NewWriter(&body)
	_, err = io.Copy(buf, resp.Body)

	if err != nil {
		return syncStatus, err
	}

	if err := json.Unmarshal(body.Bytes(), &syncStatus); err != nil {
		loghelper.LogEndpoint(bcSync).WithFields(log.Fields{
			"rawMessage": body.String(),
			"err":        err,
		}).Error("Unable to unmarshal sync status")
		return syncStatus, err
	}
	return syncStatus, nil
}

// The response when checking the lighthouse nodes DB info: /lighthouse/database/info
type LighthouseDatabaseInfo struct {
	SchemaVersion int        `json:"schema_version"`
	Config        LhDbConfig `json:"config"`
	Split         LhDbSplit  `json:"split"`
	Anchor        LhDbAnchor `json:"anchor"`
}

// The config field within the DatabaseInfo response.
type LhDbConfig struct {
	SlotsPerRestorePoint              int  `json:"slots_per_restore_point"`
	SlotsPerRestorePointSetExplicitly bool `json:"slots_per_restore_point_set_explicitly"`
	BlockCacheSize                    int  `json:"block_cache_size"`
	CompactOnInit                     bool `json:"compact_on_init"`
	CompactOnPrune                    bool `json:"compact_on_prune"`
}

// The split field within the DatabaseInfo response.
type LhDbSplit struct {
	Slot      string `json:"slot"`
	StateRoot string `json:"state_root"`
}

// The anchor field within the DatabaseInfo response.
type LhDbAnchor struct {
	AnchorSlot        string `json:"anchor_slot"`
	OldestBlockSlot   string `json:"oldest_block_slot"`
	OldestBlockParent string `json:"oldest_block_parent"`
	StateUpperLimit   string `json:"state_upper_limit"`
	StateLowerLimit   string `json:"state_lower_limit"`
}

// This function will notify us what the head slot is.
func (bc BeaconClient) queryHeadSlotInBeaconServer() (int, error) {
	syncStatus, err := bc.QueryHeadSync()
	if err != nil {
		return 0, nil
	}
	headSlot, err := strconv.Atoi(syncStatus.Data.HeadSlot)
	if err != nil {
		return 0, nil
	}
	return headSlot, nil
}

// return the lighthouse Database Info
func (bc BeaconClient) queryLighthouseDbInfo() (LighthouseDatabaseInfo, error) {
	var dbInfo LighthouseDatabaseInfo

	lhDbInfo := bc.ServerEndpoint + LhDbInfoEndpoint
	resp, err := http.Get(lhDbInfo)

	if err != nil {
		loghelper.LogEndpoint(lhDbInfo).Error("Unable to get the lighthouse database information")
		return dbInfo, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		loghelper.LogEndpoint(lhDbInfo).WithFields(log.Fields{"returnCode": resp.StatusCode}).Error("Error when getting the lighthouse database information")
		return dbInfo, fmt.Errorf("Querying the lighthouse database information returned a non 2xx status code, code provided: %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	var body bytes.Buffer
	buf := bufio.NewWriter(&body)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return dbInfo, err
	}

	if err := json.Unmarshal(body.Bytes(), &dbInfo); err != nil {
		loghelper.LogEndpoint(lhDbInfo).WithFields(log.Fields{
			"rawMessage": body.String(),
			"err":        err,
		}).Error("Unable to unmarshal the lighthouse database information")
		return dbInfo, err
	}
	return dbInfo, nil
}

// This function will tell us what the latest slot is that the beacon server has available. This is important as
// it will ensure us that we have all slots prior to the given slot.
func (bc BeaconClient) GetLatestSlotInBeaconServer(beaconServerType string) (int, error) {
	switch strings.ToLower(beaconServerType) {
	case "lighthouse":
		headSlot, err := bc.queryHeadSlotInBeaconServer()
		if err != nil {
			return 0, err
		}
		lhDb, err := bc.queryLighthouseDbInfo()
		if err != nil {
			return 0, err
		}
		if lhDb.Anchor == (LhDbAnchor{}) {
			//atomic.StoreInt64(&bc.LatestSlotInBeaconServer, int64(headSlot))
			log.WithFields(log.Fields{
				"headSlot": headSlot,
			}).Info("Anchor is nil, the lighthouse client has all the nodes from genesis to head.")
			return headSlot, nil
		} else {
			log.WithFields(log.Fields{
				"lhDb.Anchor": lhDb.Anchor,
			}).Info(LighthouseMissingSlots.Error())
			log.Info("We will add a feature down the road to wait for anchor to be null, if its needed.")
			return 0, LighthouseMissingSlots
		}
	default:
		log.WithFields(log.Fields{"BeaconServerType": beaconServerType}).Error(MissingBeaconServerType.Error())
		return 0, MissingBeaconServerType
	}
}

// A wrapper function for updating the latest slot.
func (bc BeaconClient) UpdateLatestSlotInBeaconServer(headSlot int64) {
	curr := atomic.LoadInt64(&bc.LatestSlotInBeaconServer)
	log.WithFields(log.Fields{
		"Previous Latest Slot": curr,
		"New Latest Slot":      headSlot,
	}).Debug("Swapping Head Slot")
	atomic.SwapInt64(&bc.LatestSlotInBeaconServer, int64(headSlot))
}
