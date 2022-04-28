package beaconclient

import (
	"context"
	"fmt"

	"github.com/r3labs/sse"
	log "github.com/sirupsen/logrus"
)

var (
	bcHealthEndpoint         = "/eth/v1/node/health"                        // Endpoint used for the healthcheck
	bcHeadTopicEndpoint      = "/eth/v1/events?topics=head"                 // Endpoint used to subscribe to the head of the chain
	bcReorgTopicEndpoint     = "/eth/v1/events?topics=chain_reorg"          // Endpoint used to subscribe to the head of the chain
	bcFinalizedTopicEndpoint = "/eth/v1/events?topics=finalized_checkpoint" // Endpoint used to subscribe to the head of the chain
	bcBlockQueryEndpoint     = "/eth/v2/beacon/blocks/"                     // Endpoint to query individual Blocks
	bcStateQueryEndpoint     = "/eth/v2/debug/beacon/states/"               // Endpoint to query individual States
)

// A struct that capture the Beacon Server that the Beacon Client will be interacting with and querying.
type BeaconClient struct {
	Context                     context.Context                 // A context generic context with multiple uses.
	ServerEndpoint              string                          // What is the endpoint of the beacon server.
	PerformHeadTracking         bool                            // Should we track head?
	PerformHistoricalProcessing bool                            // Should we perform historical processing?
	HeadTracking                *SseEvents[Head]                // Track the head block
	ReOrgTracking               *SseEvents[ChainReorg]          // Track all Reorgs
	FinalizationTracking        *SseEvents[FinalizedCheckpoint] // Track all finalization checkpoints
}

// A struct to keep track of relevant the head event topic.
type SseEvents[P ProcessedEvents] struct {
	Endpoint   string          // The endpoint for the subscription. Primarily used for logging
	MessagesCh chan *sse.Event // Contains all the messages from the SSE Channel
	ErrorCh    chan *SseError  // Contains any errors while SSE streaming occurred
	ProcessCh  chan *P         // Used to capture processed data in its proper struct.
	SseClient  *sse.Client     // sse.Client object that is used to interact with the SSE stream
}

// An object to capture any errors when turning an SSE message to JSON.
type SseError struct {
	err error
	msg []byte
}

// A Function to create the BeaconClient.
func CreateBeaconClient(ctx context.Context, connectionProtocol string, bcAddress string, bcPort int) *BeaconClient {
	endpoint := fmt.Sprintf("%s://%s:%d", connectionProtocol, bcAddress, bcPort)
	log.Info("Creating the BeaconClient")
	return &BeaconClient{
		Context:              ctx,
		ServerEndpoint:       endpoint,
		HeadTracking:         createSseEvent[Head](endpoint, bcHeadTopicEndpoint),
		ReOrgTracking:        createSseEvent[ChainReorg](endpoint, bcReorgTopicEndpoint),
		FinalizationTracking: createSseEvent[FinalizedCheckpoint](endpoint, bcFinalizedTopicEndpoint),
	}
}

// Create all the channels to handle a SSE events
func createSseEvent[P ProcessedEvents](baseEndpoint string, path string) *SseEvents[P] {
	endpoint := baseEndpoint + path
	sseEvents := &SseEvents[P]{
		Endpoint:   endpoint,
		MessagesCh: make(chan *sse.Event, 1),
		ErrorCh:    make(chan *SseError),
		ProcessCh:  make(chan *P),
		SseClient: func(endpoint string) *sse.Client {
			log.WithFields(log.Fields{"endpoint": endpoint}).Info("Creating SSE client")
			return sse.NewClient(endpoint)
		}(endpoint),
	}
	return sseEvents
}
