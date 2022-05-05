// This file will contain functions to query the Beacon Chain Server.

package beaconclient

import (
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

// A helper function to query endpoints that utilize slots.
func querySsz(endpoint string, slot string) ([]byte, int, error) {
	log.WithFields(log.Fields{"endpoint": endpoint}).Info("Querying endpoint")
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
