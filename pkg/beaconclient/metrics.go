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
	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-eth-beacon-indexer/pkg/loghelper"
)

// Create a metric struct and register each channel with prometheus
func CreateBeaconClientMetrics() (*BeaconClientMetrics, error) {
	metrics := &BeaconClientMetrics{
		SlotInserts:             0,
		ReorgInserts:            0,
		KnownGapsInserts:        0,
		KnownGapsProcessed:      0,
		KnownGapsReprocessError: 0,
		HeadError:               0,
		HeadReorgError:          0,
	}
	err := prometheusRegisterHelper("slot_inserts", "Keeps track of the number of slots we have inserted.", &metrics.SlotInserts)
	if err != nil {
		return nil, err
	}
	err = prometheusRegisterHelper("reorg_inserts", "Keeps track of the number of reorgs we have inserted.", &metrics.ReorgInserts)
	if err != nil {
		return nil, err
	}
	err = prometheusRegisterHelper("known_gaps_inserts", "Keeps track of the number of known gaps we have inserted.", &metrics.KnownGapsInserts)
	if err != nil {
		return nil, err
	}
	err = prometheusRegisterHelper("known_gaps_reprocess_error", "Keeps track of the number of known gaps that had errors when reprocessing, but the error was updated successfully.", &metrics.KnownGapsReprocessError)
	if err != nil {
		return nil, err
	}
	err = prometheusRegisterHelper("known_gaps_processed", "Keeps track of the number of known gaps we successfully processed.", &metrics.KnownGapsProcessed)
	if err != nil {
		return nil, err
	}
	err = prometheusRegisterHelper("historic_slots_processed", "Keeps track of the number of historic slots we successfully processed.", &metrics.HistoricSlotProcessed)
	if err != nil {
		return nil, err
	}
	err = prometheusRegisterHelper("head_error", "Keeps track of the number of errors we had processing head messages.", &metrics.HeadError)
	if err != nil {
		return nil, err
	}
	err = prometheusRegisterHelper("head_reorg_error", "Keeps track of the number of errors we had processing reorg messages.", &metrics.HeadReorgError)
	if err != nil {
		return nil, err
	}
	return metrics, nil
}

func prometheusRegisterHelper(name string, help string, varPointer *uint64) error {
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
		return err
	}
	return nil
}

// A structure utilized for keeping track of various metrics. Currently, mostly used in testing.
type BeaconClientMetrics struct {
	SlotInserts             uint64 // Number of head events we successfully wrote to the DB.
	ReorgInserts            uint64 // Number of reorg events we successfully wrote to the DB.
	KnownGapsInserts        uint64 // Number of known_gaps we successfully wrote to the DB.
	KnownGapsProcessed      uint64 // Number of knownGaps processed.
	KnownGapsReprocessError uint64 // Number of knownGaps that were updated with an error.
	HistoricSlotProcessed   uint64 // Number of historic slots successfully processed.
	HeadError               uint64 // Number of errors that occurred when decoding the head message.
	HeadReorgError          uint64 // Number of errors that occurred when decoding the reorg message.
}

// Wrapper function to increment inserts. If we want to use mutexes later we can easily update all
// occurrences here.
func (m *BeaconClientMetrics) IncrementSlotInserts(inc uint64) {
	log.Debug("Incrementing Slot Insert")
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
	atomic.AddUint64(&m.KnownGapsProcessed, inc)
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

// Wrapper function to increment the number of knownGaps that were updated with reprocessing errors.
// If we want to use mutexes later we can easily update all occurrences here.
func (m *BeaconClientMetrics) IncrementKnownGapsReprocessError(inc uint64) {
	atomic.AddUint64(&m.KnownGapsReprocessError, inc)
}

// Wrapper function to increment the number of historicSlots that were processed successfully.
// If we want to use mutexes later we can easily update all occurrences here.
func (m *BeaconClientMetrics) IncrementHistoricSlotProcessed(inc uint64) {
	atomic.AddUint64(&m.HistoricSlotProcessed, inc)
}
