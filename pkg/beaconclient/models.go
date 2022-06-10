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

// This interface captured what the events can be for processed event streams.
type ProcessedEvents interface {
	Head | ChainReorg
}

// This struct captures the JSON representation of the head topic
type Head struct {
	Slot                      string `json:"slot"`
	Block                     string `json:"block"`
	State                     string `json:"state"`
	CurrentDutyDependentRoot  string `json:"current_duty_dependent_root"`
	PreviousDutyDependentRoot string `json:"previous_duty_dependent_root"`
	EpochTransition           bool   `json:"epoch_transition"`
	ExecutionOptimistic       bool   `json:"execution_optimistic"`
}

// This struct captures the JSON representation of the finalized_checkpoint topic.
type FinalizedCheckpoint struct {
	Block               string `json:"block"`
	State               string `json:"state"`
	Epoch               string `json:"epoch"`
	ExecutionOptimistic bool   `json:"execution_optimistic"`
}

// This struct captures the JSON representation of the chain_reorg topic.
type ChainReorg struct {
	Slot                string `json:"slot"`
	Depth               string `json:"depth"`
	OldHeadBlock        string `json:"old_head_block"`
	NewHeadBlock        string `json:"new_head_block"`
	OldHeadState        string `json:"old_head_state"`
	NewHeadState        string `json:"new_head_state"`
	Epoch               string `json:"epoch"`
	ExecutionOptimistic bool   `json:"execution_optimistic"`
}

// A struct to capture whats being written to the eth-beacon.slots table.
type DbSlots struct {
	Epoch     string // The epoch.
	Slot      string // The slot.
	BlockRoot string // The block root
	StateRoot string // The state root
	Status    string // The status, it can be proposed | forked | skipped.
}

// A struct to capture whats being written to eth-beacon.signed_block table.
type DbSignedBeaconBlock struct {
	Slot          string // The slot.
	BlockRoot     string // The block root
	ParentBlock   string // The parent block root.
	Eth1BlockHash string // The eth1 block_hash
	MhKey         string // The ipld multihash key.

}

// A struct to capture whats being written to eth-beacon.state table.
type DbBeaconState struct {
	Slot      string // The slot.
	StateRoot string // The state root
	MhKey     string // The ipld multihash key.
}

// A structure to capture whats being written to the eth-beacon.known_gaps table.
type DbKnownGaps struct {
	StartSlot         string // The start slot for known_gaps, inclusive.
	EndSlot           string // The end slot for known_gaps, inclusive.
	CheckedOut        bool   // Indicates if any process is currently processing this entry.
	ReprocessingError string // The error that occurred when attempting to reprocess these entries.
	EntryError        string // The error that caused this entry to be added to the table. Could be null.
	EntryTime         string // The time this range was added to the DB. This can help us catch ranges that have not been processed for a long time due to some error.
	EntryProcess      string // The entry process that added this process. Potential options are StartUp, Error, ManualEntry, HeadGap.
}
