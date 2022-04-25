package healthcheck

import (
	"fmt"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

var (
	bcHealthEndpoint = "/eth/v1/node/health"
)

// This function will ensure that we can connect to the beacon client.
// Keep in mind, the beacon client will allow you to connect to it but it might
// Not allow you to make http requests. This is part of its built in logic, and you will have
// to follow their provided guidelines. https://lighthouse-book.sigmaprime.io/api-bn.html#security
func CheckBeaconClient(bcAddress string, bcPort int) error {
	log.Debug("Attempting to connect to the beacon client")
	bcEndpoint := "http://" + bcAddress + ":" + strconv.Itoa(bcPort) + bcHealthEndpoint
	resp, err := http.Get(bcEndpoint)
	if err != nil {
		loghelper.LogError(err).Error("Unable to get bc endpoint: ", bcEndpoint)
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Error("We recieved a non 2xx status code when checking the health of the beacon node.")
		log.Error("Health Endpoint Status Code: ", resp.StatusCode)
		return fmt.Errorf("beacon Node Provided a non 2xx status code, code provided: %d", resp.StatusCode)
	}

	log.Info("We can successfully reach the beacon client.")
	return nil

}
