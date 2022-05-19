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
	"sync/atomic"
)

// A structure utilized for keeping track of various metrics. Currently, mostly used in testing.
type BeaconClientMetrics struct {
	HeadTrackingInserts   uint64 // Number of head events we successfully wrote to the DB.
	HeadTrackingReorgs    uint64 // Number of reorg events we successfully wrote to the DB.
	HeadTrackingKnownGaps uint64 // Number of known_gaps we successfully wrote to the DB.
	HeadError             uint64 // Number of errors that occurred when decoding the head message.
	HeadReorgError        uint64 // Number of errors that occurred when decoding the reorg message.
}

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
