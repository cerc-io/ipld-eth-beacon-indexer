// This file will contain functions to query the Beacon Chain Server.

package beaconclient

import (
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

// Attempt to use generics..
// // These are types that append slot at the end of the URL to handle a request.
// type SlotBasedRequests interface {
// 	*specs.BeaconState | *specs.SignedBeaconBlock
// 	UnmarshalSSZ([]byte) error
// }
//
// func queryState[R SlotBasedRequests](endpoint string, slot string) (R, error) {
// 	obj := new(R)
// 	rawState, err := querySlot(endpoint, slot)
// 	if err != nil {
// 		return *obj, err
// 	}
//
// 	err = &obj.UnmarshalSSZ(rawState)
// 	err = (*obj).UnmarshalSSZ(rawState)
// 	if err != nil {
// 		loghelper.LogSlotError(slot, err).Error("Unable to unmarshal the SSZ response from the Beacon Node Successfully!")
// 		return *obj, fmt.Errorf("Unable to unmarshal the SSZ response from the Beacon Node Successfully!: %s", err.Error())
// 	}
// 	return *obj, nil
// }

// This function will query a state object based on the slot provided.
// The object is SSZ encoded.

//type BeaconBlockResponse struct {
//	version string `json: `
//}

// func queryState(endpoint string, slot string) (spectests.BeaconState, error) {
// 	obj := new(spectests.BeaconState)
// 	fullEndpoint := endpoint + slot
// 	rawState, err := querySsz(fullEndpoint, slot)
// 	if err != nil {
// 		return *obj, err
// 	}
//
// 	err = obj.UnmarshalSSZ(rawState)
// 	if err != nil {
// 		loghelper.LogSlotError(slot, err).Error("Unable to unmarshal the SSZ response from the Beacon Node")
// 		return *obj, fmt.Errorf("Unable to unmarshal the SSZ response from the Beacon Node: %s", err.Error())
// 	}
// 	return *obj, nil
// }
//
// // This function will query a state object based on the slot provided.
// // The object is SSZ encoded.
// func queryBlock(endpoint string, slot string) (spectests.SignedBeaconBlock, error) {
// 	obj := new(spectests.SignedBeaconBlock)
// 	fullEndpoint := endpoint + slot
// 	rawBlock, err := querySsz(fullEndpoint, slot)
// 	if err != nil {
// 		return *obj, err
// 	}
//
// 	err = obj.UnmarshalSSZ(rawBlock)
// 	if err != nil {
// 		loghelper.LogSlotError(slot, err).Error("Unable to unmarshal the SSZ response from the Beacon Node Successfully!")
// 		return *obj, fmt.Errorf("Unable to unmarshal the SSZ response from the Beacon Node Successfully!: %s", err.Error())
// 	}
// 	return *obj, nil
// }

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
