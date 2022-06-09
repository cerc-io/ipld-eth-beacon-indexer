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
// This file will contain functions to query the Beacon Chain Server.

package beaconclient

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/loghelper"
)

// A helper function to query endpoints that utilize slots.
func querySsz(endpoint string, slot string) ([]byte, int, error) {
	log.WithFields(log.Fields{"endpoint": endpoint}).Debug("Querying endpoint")
	client := &http.Client{}
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		loghelper.LogSlotError(slot, err).Error("Unable to create a request!")
		return nil, 0, fmt.Errorf("Unable to create a request!: %s", err.Error())
	}
	req.Header.Set("Accept", "application/octet-stream")
	response, err := client.Do(req)
	if err != nil {
		loghelper.LogSlotError(slot, err).Error("Unable to query Beacon Node!")
		return nil, 0, fmt.Errorf("Unable to query Beacon Node: %s", err.Error())
	}
	defer response.Body.Close()
	rc := response.StatusCode
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		loghelper.LogSlotError(slot, err).Error("Unable to turn response into a []bytes array!")
		return nil, rc, fmt.Errorf("Unable to turn response into a []bytes array!: %s", err.Error())
	}
	return body, rc, nil
}

// Object to unmarshal the BlockRootResponse
type BlockRootResponse struct {
	Data BlockRootMessage `json:"data"`
}

// Object to unmarshal the BlockRoot Message
type BlockRootMessage struct {
	Root string `json:"root"`
}

// A function to query the blockroot for a given slot.
func queryBlockRoot(endpoint string, slot string) (string, error) {
	log.WithFields(log.Fields{"endpoint": endpoint}).Debug("Querying endpoint")
	client := &http.Client{}
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		loghelper.LogSlotError(slot, err).Error("Unable to create a request!")
		return "", fmt.Errorf("Unable to create a request!: %s", err.Error())
	}
	req.Header.Set("Accept", "application/json")
	response, err := client.Do(req)
	if err != nil {
		loghelper.LogSlotError(slot, err).Error("Unable to query Beacon Node!")
		return "", fmt.Errorf("Unable to query Beacon Node: %s", err.Error())
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		loghelper.LogSlotError(slot, err).Error("Unable to turn response into a []bytes array!")
		return "", fmt.Errorf("Unable to turn response into a []bytes array!: %s", err.Error())
	}

	resp := BlockRootResponse{}
	if err := json.Unmarshal(body, &resp); err != nil {
		loghelper.LogEndpoint(endpoint).WithFields(log.Fields{
			"rawMessage": string(body),
			"err":        err,
		}).Error("Unable to unmarshal the block root")
		return "", err
	}
	return resp.Data.Root, nil
}
