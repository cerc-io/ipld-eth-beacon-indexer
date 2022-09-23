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
	"context"
	"fmt"
	"github.com/r3labs/sse/v2"
	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/database/sql"
	"math/rand"
	"time"
)

// TODO: Use prysms config values instead of hardcoding them here.
var (
	bcHealthEndpoint     = "/eth/v1/node/health"               // Endpoint used for the healthcheck
	BcHeadTopicEndpoint  = "/eth/v1/events?topics=head"        // Endpoint used to subscribe to the head of the chain
	bcReorgTopicEndpoint = "/eth/v1/events?topics=chain_reorg" // Endpoint used to subscribe to the head of the chain
	BcBlockQueryEndpoint = "/eth/v2/beacon/blocks/"            // Endpoint to query individual Blocks
	BcStateQueryEndpoint = "/eth/v2/debug/beacon/states/"      // Endpoint to query individual States
	BcSyncStatusEndpoint = "/eth/v1/node/syncing"              // The endpoint to check to see if the beacon server is still trying to sync to head.
	LhDbInfoEndpoint     = "/lighthouse/database/info"         // The endpoint for the LIGHTHOUSE server to get the database information.
	BcBlockRootEndpoint  = func(slot string) string {
		return "/eth/v1/beacon/blocks/" + slot + "/root"
	}
	bcSlotsPerEpoch uint64 = 32 // Number of slots in a single Epoch
	//bcSlotPerHistoricalVector = 8192                                // The number of slots in a historic vector.
	//bcFinalizedTopicEndpoint  = "/eth/v1/events?topics=finalized_checkpoint" // Endpoint used to subscribe to the head of the chain
)

// A struct that capture the Beacon Server that the Beacon Client will be interacting with and querying.
type BeaconClient struct {
	Context                      context.Context      // A context generic context with multiple uses.
	ServerEndpoint               string               // What is the endpoint of the beacon server.
	Db                           sql.Database         // Database object used for reads and writes.
	Metrics                      *BeaconClientMetrics // An object used to keep track of certain BeaconClient Metrics.
	KnownGapTableIncrement       int                  // The max number of slots within a single known_gaps table entry.
	UniqueNodeIdentifier         int                  // The unique identifier within the cluster of this individual node.
	KnownGapsProcess             KnownGapsProcessing  // object keeping track of knowngaps processing
	CheckDb                      bool                 // Should we check the DB to see if the slot exists before processing it?
	PerformBeaconStateProcessing bool                 // Should we process BeaconStates?
	PerformBeaconBlockProcessing bool                 // Should we process BeaconBlocks?

	// Used for Head Tracking

	PerformHeadTracking bool                   // Should we track head?
	StartingSlot        uint64                 // If we're performing head tracking. What is the first slot we processed.
	PreviousSlot        uint64                 // Whats the previous slot we processed
	PreviousBlockRoot   string                 // Whats the previous block root, used to check the next blocks parent.
	HeadTracking        *SseEvents[Head]       // Track the head block
	ReOrgTracking       *SseEvents[ChainReorg] // Track all Reorgs
	//FinalizationTracking        *SseEvents[FinalizedCheckpoint] // Track all finalization checkpoints

	// Used for Historical Processing

	// The latest available slot within the Beacon Server. We can't query any slot greater than this.
	// This value is lazily updated. Therefore at times it will be outdated.
	LatestSlotInBeaconServer    int64
	PerformHistoricalProcessing bool               // Should we perform historical processing?
	HistoricalProcess           HistoricProcessing // object keeping track of historical processing
}

// A struct to keep track of relevant the head event topic.
type SseEvents[P ProcessedEvents] struct {
	Endpoint   string          // The endpoint for the subscription. Primarily used for logging
	MessagesCh chan *sse.Event // Contains all the messages from the SSE Channel
	ErrorCh    chan *SseError  // Contains any errors while SSE streaming occurred
	ProcessCh  chan *P         // Used to capture processed data in its proper struct.
	sseClient  *sse.Client     // sse.Client object that is used to interact with the SSE stream
}

// An object to capture any errors when turning an SSE message to JSON.
type SseError struct {
	err error
	msg []byte
}

// A Function to create the BeaconClient.
func CreateBeaconClient(ctx context.Context, connectionProtocol string, bcAddress string, bcPort int,
	bcKgTableIncrement int, uniqueNodeIdentifier int, checkDb bool, performBeaconBlockProcessing bool, performBeaconStateProcessing bool) (*BeaconClient, error) {
	if uniqueNodeIdentifier == 0 {
		uniqueNodeIdentifier := rand.Int()
		log.WithField("randomUniqueNodeIdentifier", uniqueNodeIdentifier).Warn("No uniqueNodeIdentifier provided, we are going to use a randomly generated one.")
	}

	metrics, err := CreateBeaconClientMetrics()
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s://%s:%d", connectionProtocol, bcAddress, bcPort)
	log.Info("Creating the BeaconClient")
	return &BeaconClient{
		Context:                      ctx,
		ServerEndpoint:               endpoint,
		KnownGapTableIncrement:       bcKgTableIncrement,
		HeadTracking:                 createSseEvent[Head](endpoint, BcHeadTopicEndpoint),
		ReOrgTracking:                createSseEvent[ChainReorg](endpoint, bcReorgTopicEndpoint),
		Metrics:                      metrics,
		UniqueNodeIdentifier:         uniqueNodeIdentifier,
		CheckDb:                      checkDb,
		PerformBeaconBlockProcessing: performBeaconBlockProcessing,
		PerformBeaconStateProcessing: performBeaconStateProcessing,
		//FinalizationTracking: createSseEvent[FinalizedCheckpoint](endpoint, bcFinalizedTopicEndpoint),
	}, nil
}

// Create all the channels to handle a SSE events
func createSseEvent[P ProcessedEvents](baseEndpoint string, path string) *SseEvents[P] {
	endpoint := baseEndpoint + path
	sseEvents := &SseEvents[P]{
		Endpoint:   endpoint,
		MessagesCh: make(chan *sse.Event, 1),
		ErrorCh:    make(chan *SseError),
		ProcessCh:  make(chan *P),
	}
	return sseEvents
}

func (se *SseEvents[P]) Connect() error {
	if nil == se.sseClient {
		se.initClient()
	}
	return se.sseClient.SubscribeChanRaw(se.MessagesCh)
}

func (se *SseEvents[P]) Disconnect() {
	if nil == se.sseClient {
		return
	}

	log.WithFields(log.Fields{"endpoint": se.Endpoint}).Info("Disconnecting and destroying SSE client")
	se.sseClient.Unsubscribe(se.MessagesCh)
	se.sseClient.Connection.CloseIdleConnections()
	se.sseClient = nil
}

func (se *SseEvents[P]) initClient() {
	if nil != se.sseClient {
		se.Disconnect()
	}

	log.WithFields(log.Fields{"endpoint": se.Endpoint}).Info("Creating SSE client")
	client := sse.NewClient(se.Endpoint)
	client.ReconnectNotify = func(err error, duration time.Duration) {
		log.WithFields(log.Fields{"endpoint": se.Endpoint}).Debug("Reconnecting SSE client")
	}
	client.OnDisconnect(func(c *sse.Client) {
		log.WithFields(log.Fields{"endpoint": se.Endpoint}).Debug("SSE client disconnected")
	})
	se.sseClient = client
}
