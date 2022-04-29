package beaconclient

import "math/big"

// This interface captured what the events can be for processed event streams.
type ProcessedEvents interface {
	Head | FinalizedCheckpoint | ChainReorg
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

// A struct to capture whats being written to the ethcl.slots table.
type DbSlots struct {
	Epoch     *big.Int // The epoch.
	Slot      *big.Int // The slot.
	BlockRoot string   // The block root
	StateRoot string   // The state root
	status    string   // The status, it can be proposed | forked | missed.
}

// A struct to capture whats being written to ethcl.signed_beacon_block table.
type DbSignedBeaconBlock struct {
	Slot        *big.Int // The slot.
	BlockRoot   string   // The block root
	ParentBlock string   // The parent block root.
	mh_key      string   // The ipld multihash key.

}

type DbBeaconState struct {
	Slot      *big.Int // The slot.
	StateRoot string   // The state root
	mh_key    string   // The ipld multihash key.
}
