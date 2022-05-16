package beaconclient

import (
	"sync/atomic"

	log "github.com/sirupsen/logrus"
)

// Wrapper function to increment inserts. If we want to use mutexes later we can easily update all
// occurrences here.
func (m *BeaconClientMetrics) IncrementHeadTrackingInserts(inc uint64) {
	log.Info("Updating the insert ")
	atomic.AddUint64(&m.HeadTrackingInserts, inc)
}

// Wrapper function to increment reorgs. If we want to use mutexes later we can easily update all
// occurrences here.
func (m *BeaconClientMetrics) IncrementHeadTrackingReorgs(inc uint64) {
	atomic.AddUint64(&m.HeadTrackingReorgs, inc)
}

// Wrapper function to increment reorgs. If we want to use mutexes later we can easily update all
// occurrences here.
func (m *BeaconClientMetrics) IncrementHeadTrackingKnownGaps(inc uint64) {
	atomic.AddUint64(&m.HeadTrackingKnownGaps, inc)
}
