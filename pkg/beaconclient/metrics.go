package beaconclient

import "sync/atomic"

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
