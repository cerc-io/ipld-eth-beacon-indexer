package beaconclient

import (
	"sync/atomic"
)

// Wrapper function to increment inserts. If we want to use mutexes later we can easily update all
// occurrences here.
func (m *BeaconClientMetrics) IncrementHeadTrackingInserts(inc uint64) {
	atomic.AddUint64(&m.HeadTrackingInserts, inc)
}

// Wrapper function to increment reorgs. If we want to use mutexes later we can easily update all
// occurrences here.
func (m *BeaconClientMetrics) IncrementHeadTrackingReorgs(inc uint64) {
	atomic.AddUint64(&m.HeadTrackingReorgs, inc)
}

// Wrapper function to increment known gaps. If we want to use mutexes later we can easily update all
// occurrences here.
func (m *BeaconClientMetrics) IncrementHeadTrackingKnownGaps(inc uint64) {
	atomic.AddUint64(&m.HeadTrackingKnownGaps, inc)
}

// Wrapper function to increment head errors. If we want to use mutexes later we can easily update all
// occurrences here.
func (m *BeaconClientMetrics) IncrementHeadError(inc uint64) {
	atomic.AddUint64(&m.HeadError, inc)
}

// Wrapper function to increment reorg errors. If we want to use mutexes later we can easily update all
// occurrences here.
func (m *BeaconClientMetrics) IncrementHeadReorgError(inc uint64) {
	atomic.AddUint64(&m.HeadReorgError, inc)
}
