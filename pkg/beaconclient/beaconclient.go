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
	connectionProtocol       = "http"
)

// A struct that capture the Beacon Server that the Beacon Client will be interacting with and querying.
type BeaconClient struct {
	Context                     context.Context                 // A context generic context with multiple uses.
	ServerAddress               string                          // Address of the Beacon Server
	ServerPort                  int                             // Port of the Beacon Server
	PerformHeadTracking         bool                            // Should we track head?
	PerformHistoricalProcessing bool                            // Should we perform historical processing?
	HeadTracking                *SseEvents[Head]                // Track the head block
	ReOrgTracking               *SseEvents[ChainReorg]          // Track all Reorgs
	FinalizationTracking        *SseEvents[FinalizedCheckpoint] // Track all finalization checkpoints
}

// A struct to keep track of relevant the head event topic.
type SseEvents[P ProcessedEvents] struct {
	Url        string          // The url for the subscription. Primarily used for logging
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
func CreateBeaconClient(ctx context.Context, bcAddress string, bcPort int) *BeaconClient {
	log.Info("Creating the BeaconClient")
	return &BeaconClient{
		Context:              ctx,
		ServerAddress:        bcAddress,
		ServerPort:           bcPort,
		HeadTracking:         createSseEvent[Head](connectionProtocol, bcAddress, bcPort, bcHeadTopicEndpoint),
		ReOrgTracking:        createSseEvent[ChainReorg](connectionProtocol, bcAddress, bcPort, bcReorgTopicEndpoint),
		FinalizationTracking: createSseEvent[FinalizedCheckpoint](connectionProtocol, bcAddress, bcPort, bcFinalizedTopicEndpoint),
	}
}

// Create all the channels to handle a SSE events
func createSseEvent[P ProcessedEvents](connectionProtocol, bcAddress string, bcPort int, endpoint string) *SseEvents[P] {
	url := fmt.Sprintf("%s://%s:%d%s", connectionProtocol, bcAddress, bcPort, endpoint)
	sseEvents := &SseEvents[P]{
		Url:        url,
		MessagesCh: make(chan *sse.Event),
		ErrorCh:    make(chan *SseError),
		ProcessCh:  make(chan *P),
		SseClient: func(url string) *sse.Client {
			log.WithFields(log.Fields{"url": url}).Info("Creating SSE client")
			return sse.NewClient(url)
		}(url),
	}
	return sseEvents
}
