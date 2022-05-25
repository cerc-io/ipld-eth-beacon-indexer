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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

//Create a metric struct and register each channel with prometheus
func CreateBeaconClientMetrics() *BeaconClientMetrics {
	metrics := &BeaconClientMetrics{
		SlotInserts:              0,
		ReorgInserts:             0,
		KnownGapsInserts:         0,
		knownGapsProcessed:       0,
		KnownGapsProcessingError: 0,
		HeadError:                0,
		HeadReorgError:           0,
	}
	prometheusRegisterHelper("slot_inserts", "Keeps track of the number of slots we have inserted.", &metrics.SlotInserts)
	prometheusRegisterHelper("reorg_inserts", "Keeps track of the number of reorgs we have inserted.", &metrics.ReorgInserts)
	prometheusRegisterHelper("known_gaps_inserts", "Keeps track of the number of known gaps we have inserted.", &metrics.KnownGapsInserts)
	prometheusRegisterHelper("known_gaps_processed", "Keeps track of the number of known gaps we processed.", &metrics.knownGapsProcessed)
	prometheusRegisterHelper("known_gaps_processing_error", "Keeps track of the number of known gaps we had errors processing.", &metrics.KnownGapsProcessingError)
	prometheusRegisterHelper("head_error", "Keeps track of the number of errors we had processing head messages.", &metrics.HeadError)
	prometheusRegisterHelper("head_reorg_error", "Keeps track of the number of errors we had processing reorg messages.", &metrics.HeadReorgError)
	return metrics
}

func prometheusRegisterHelper(name string, help string, varPointer *uint64) {
	err := prometheus.Register(prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Namespace:   "beacon_client",
			Subsystem:   "",
			Name:        name,
			Help:        help,
			ConstLabels: map[string]string{},
		},
		func() float64 {
			return float64(atomic.LoadUint64(varPointer))
		}))
	if err != nil && err.Error() != "duplicate metrics collector registration attempted" {
		loghelper.LogError(err).WithField("name", name).Error("Unable to register counter.")
	}
}

// A structure utilized for keeping track of various metrics. Currently, mostly used in testing.
type BeaconClientMetrics struct {
	SlotInserts              uint64 // Number of head events we successfully wrote to the DB.
	ReorgInserts             uint64 // Number of reorg events we successfully wrote to the DB.
	KnownGapsInserts         uint64 // Number of known_gaps we successfully wrote to the DB.
	knownGapsProcessed       uint64 // Number of knownGaps processed.
	KnownGapsProcessingError uint64 // Number of errors that occurred while processing a knownGap
	HeadError                uint64 // Number of errors that occurred when decoding the head message.
	HeadReorgError           uint64 // Number of errors that occurred when decoding the reorg message.
}

// Wrapper function to increment inserts. If we want to use mutexes later we can easily update all
// occurrences here.
func (m *BeaconClientMetrics) IncrementSlotInserts(inc uint64) {
	logrus.Debug("Incrementing Slot Insert")
	atomic.AddUint64(&m.SlotInserts, inc)
}

// Wrapper function to increment reorgs. If we want to use mutexes later we can easily update all
// occurrences here.
func (m *BeaconClientMetrics) IncrementReorgsInsert(inc uint64) {
	atomic.AddUint64(&m.ReorgInserts, inc)
}

// Wrapper function to increment known gaps. If we want to use mutexes later we can easily update all
// occurrences here.
func (m *BeaconClientMetrics) IncrementKnownGapsInserts(inc uint64) {
	atomic.AddUint64(&m.KnownGapsInserts, inc)
}

// Wrapper function to increment known gaps processed. If we want to use mutexes later we can easily update all
// occurrences here.
func (m *BeaconClientMetrics) IncrementKnownGapsProcessed(inc uint64) {
	atomic.AddUint64(&m.knownGapsProcessed, inc)
}

// Wrapper function to increment known gaps processing error. If we want to use mutexes later we can easily update all
// occurrences here.
func (m *BeaconClientMetrics) IncrementKnownGapsProcessingError(inc uint64) {
	atomic.AddUint64(&m.KnownGapsProcessingError, inc)
}

// Wrapper function to increment head errors. If we want to use mutexes later we can easily update all
// occurrences here.
func (m *BeaconClientMetrics) IncrementHeadError(inc uint64) {
	atomic.AddUint64(&m.HeadError, inc)
}

// Wrapper function to increment reorg errors. If we want to use mutexes later we can easily update all
// occurrences here.
func (m *BeaconClientMetrics) IncrementReorgError(inc uint64) {
	atomic.AddUint64(&m.HeadReorgError, inc)
}
