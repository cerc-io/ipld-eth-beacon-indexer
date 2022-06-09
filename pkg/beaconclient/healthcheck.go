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
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/loghelper"
)

// This function will ensure that we can connect to the beacon client.
// Keep in mind, the beacon client will allow you to connect to it but it might
// Not allow you to make http requests. This is part of its built in logic, and you will have
// to follow their provided guidelines. https://lighthouse-book.sigmaprime.io/api-bn.html#security
func (bc BeaconClient) CheckBeaconClient() error {
	log.Debug("Attempting to connect to the beacon client")
	bcEndpoint := bc.ServerEndpoint + bcHealthEndpoint
	resp, err := http.Get(bcEndpoint)
	if err != nil {
		loghelper.LogError(err).Error("Unable to get bc endpoint: ", bcEndpoint)
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		loghelper.LogEndpoint(bcEndpoint).Error("We recieved a non 2xx status code when checking the health of the beacon node.")
		loghelper.LogEndpoint(bcEndpoint).Error("Health Endpoint Status Code: ", resp.StatusCode)
		return fmt.Errorf("beacon Node Provided a non 2xx status code, code provided: %d", resp.StatusCode)
	}

	log.Info("We can successfully reach the beacon client.")
	return nil

}
