package beaconclient

import (
	"errors"
	log "github.com/sirupsen/logrus"
	consensus "github.com/umbracle/go-eth-consensus"
)

type SignedBeaconBlock struct {
	signedBeaconBlockBellatrix *consensus.SignedBeaconBlockBellatrix
	signedBeaconBlockAltair    *consensus.SignedBeaconBlockAltair
	signedBeaconBlockPhase0    *consensus.SignedBeaconBlockPhase0
}

type BeaconBlock struct {
	beaconBlockBellatrix *consensus.BeaconBlockBellatrix
	beaconBlockAltair    *consensus.BeaconBlockAltair
	beaconBlockPhase0    *consensus.BeaconBlockPhase0
}

type BeaconBlockBody struct {
	beaconBlockBodyBellatrix *consensus.BeaconBlockBodyBellatrix
	beaconBlockBodyAltair    *consensus.BeaconBlockBodyAltair
	beaconBlockBodyPhase0    *consensus.BeaconBlockBodyPhase0
}

type BeaconState struct {
	beaconStateBellatrix *consensus.BeaconStateBellatrix
	beaconStateAltair    *consensus.BeaconStateAltair
	beaconStatePhase0    *consensus.BeaconStatePhase0
}

func (s *SignedBeaconBlock) UnmarshalSSZ(ssz []byte) error {
	var bellatrix consensus.SignedBeaconBlockBellatrix
	err := bellatrix.UnmarshalSSZ(ssz)
	if nil == err {
		s.signedBeaconBlockBellatrix = &bellatrix
		s.signedBeaconBlockAltair = nil
		s.signedBeaconBlockPhase0 = nil
		log.Info("Unmarshalled Bellatrix SignedBeaconBlock")
		return nil
	}

	var altair consensus.SignedBeaconBlockAltair
	err = altair.UnmarshalSSZ(ssz)
	if nil == err {
		s.signedBeaconBlockBellatrix = nil
		s.signedBeaconBlockAltair = &altair
		s.signedBeaconBlockPhase0 = nil
		log.Info("Unmarshalled Altair SignedBeaconBlock")
		return nil
	}

	var phase0 consensus.SignedBeaconBlockPhase0
	err = phase0.UnmarshalSSZ(ssz)
	if nil == err {
		s.signedBeaconBlockBellatrix = nil
		s.signedBeaconBlockAltair = nil
		s.signedBeaconBlockPhase0 = &phase0
		log.Info("Unmarshalled Phase0 SignedBeaconBlock")
		return nil
	}

	s.signedBeaconBlockBellatrix = nil
	s.signedBeaconBlockAltair = nil
	s.signedBeaconBlockPhase0 = nil

	log.Warning("Unable to unmarshal SignedBeaconBlock")
	return err
}

func (s *SignedBeaconBlock) MarshalSSZ() ([]byte, error) {
	if s.IsBellatrix() {
		return s.signedBeaconBlockBellatrix.MarshalSSZ()
	}
	if s.IsAltair() {
		return s.signedBeaconBlockAltair.MarshalSSZ()
	}
	if s.IsPhase0() {
		return s.signedBeaconBlockPhase0.MarshalSSZ()
	}

	return []byte{}, errors.New("SignedBeaconBlock not set")
}

func (s *SignedBeaconBlock) IsBellatrix() bool {
	return s.signedBeaconBlockBellatrix != nil
}

func (s *SignedBeaconBlock) IsAltair() bool {
	return s.signedBeaconBlockAltair != nil
}

func (s *SignedBeaconBlock) IsPhase0() bool {
	return s.signedBeaconBlockPhase0 != nil
}

func (s *SignedBeaconBlock) GetBellatrix() *consensus.SignedBeaconBlockBellatrix {
	return s.signedBeaconBlockBellatrix
}

func (s *SignedBeaconBlock) GetAltair() *consensus.SignedBeaconBlockAltair {
	return s.signedBeaconBlockAltair
}

func (s *SignedBeaconBlock) GetPhase0() *consensus.SignedBeaconBlockPhase0 {
	return s.signedBeaconBlockPhase0
}

func (s *SignedBeaconBlock) Signature() *consensus.Signature {
	if s.IsBellatrix() {
		return &s.signedBeaconBlockBellatrix.Signature
	}

	if s.IsAltair() {
		return &s.signedBeaconBlockAltair.Signature
	}

	if s.IsPhase0() {
		return &s.signedBeaconBlockPhase0.Signature
	}

	return nil
}

func (s *SignedBeaconBlock) Block() *BeaconBlock {
	if s.IsBellatrix() {
		return &BeaconBlock{beaconBlockBellatrix: s.signedBeaconBlockBellatrix.Block}
	}

	if s.IsAltair() {
		return &BeaconBlock{beaconBlockAltair: s.signedBeaconBlockAltair.Block}
	}

	if s.IsPhase0() {
		return &BeaconBlock{beaconBlockPhase0: s.signedBeaconBlockPhase0.Block}
	}

	return nil
}

func (b *BeaconBlock) IsBellatrix() bool {
	return b.beaconBlockBellatrix != nil
}

func (b *BeaconBlock) IsAltair() bool {
	return b.beaconBlockAltair != nil
}

func (b *BeaconBlock) IsPhase0() bool {
	return b.beaconBlockPhase0 != nil
}

func (s *BeaconBlock) GetBellatrix() *consensus.BeaconBlockBellatrix {
	return s.beaconBlockBellatrix
}

func (s *BeaconBlock) GetAltair() *consensus.BeaconBlockAltair {
	return s.beaconBlockAltair
}

func (s *BeaconBlock) GetPhase0() *consensus.BeaconBlockPhase0 {
	return s.beaconBlockPhase0
}

func (b *BeaconBlock) ParentRoot() *consensus.Root {
	if b.IsBellatrix() {
		return &b.beaconBlockBellatrix.ParentRoot
	}

	if b.IsAltair() {
		return &b.beaconBlockAltair.ParentRoot
	}

	if b.IsPhase0() {
		return &b.beaconBlockPhase0.ParentRoot
	}

	return nil
}

func (b *BeaconBlock) StateRoot() *consensus.Root {
	if b.IsBellatrix() {
		return &b.beaconBlockBellatrix.StateRoot
	}

	if b.IsAltair() {
		return &b.beaconBlockAltair.StateRoot
	}

	if b.IsPhase0() {
		return &b.beaconBlockPhase0.StateRoot
	}

	return nil
}

func (b *BeaconBlock) Body() *BeaconBlockBody {
	if b.IsBellatrix() {
		return &BeaconBlockBody{beaconBlockBodyBellatrix: b.beaconBlockBellatrix.Body}
	}

	if b.IsAltair() {
		return &BeaconBlockBody{beaconBlockBodyAltair: b.beaconBlockAltair.Body}
	}

	if b.IsPhase0() {
		return &BeaconBlockBody{beaconBlockBodyPhase0: b.beaconBlockPhase0.Body}
	}

	return nil
}

func (b *BeaconBlockBody) IsBellatrix() bool {
	return b.beaconBlockBodyBellatrix != nil
}

func (b *BeaconBlockBody) IsAltair() bool {
	return b.beaconBlockBodyAltair != nil
}

func (b *BeaconBlockBody) IsPhase0() bool {
	return b.beaconBlockBodyPhase0 != nil
}

func (b *BeaconBlockBody) Eth1Data() *consensus.Eth1Data {
	if b.IsBellatrix() {
		return b.beaconBlockBodyBellatrix.Eth1Data
	}

	if b.IsAltair() {
		return b.beaconBlockBodyAltair.Eth1Data
	}

	if b.IsPhase0() {
		return b.beaconBlockBodyPhase0.Eth1Data
	}

	return nil
}

func (b *BeaconBlock) HashTreeRoot() ([32]byte, error) {
	if b.IsBellatrix() {
		return b.beaconBlockBellatrix.HashTreeRoot()
	}

	if b.IsAltair() {
		return b.beaconBlockAltair.HashTreeRoot()
	}

	if b.IsPhase0() {
		return b.beaconBlockPhase0.HashTreeRoot()
	}

	return [32]byte{}, errors.New("BeaconBlock not set")
}

func (s *BeaconState) UnmarshalSSZ(ssz []byte) error {
	var bellatrix consensus.BeaconStateBellatrix
	err := bellatrix.UnmarshalSSZ(ssz)
	if nil == err {
		s.beaconStateBellatrix = &bellatrix
		s.beaconStateAltair = nil
		s.beaconStatePhase0 = nil
		log.Info("Unmarshalled Bellatrix BeaconState")
		return nil
	}

	var altair consensus.BeaconStateAltair
	err = altair.UnmarshalSSZ(ssz)
	if nil == err {
		s.beaconStateBellatrix = nil
		s.beaconStateAltair = &altair
		s.beaconStatePhase0 = nil
		log.Info("Unmarshalled Altair BeaconState")
		return nil
	}

	var phase0 consensus.BeaconStatePhase0
	err = phase0.UnmarshalSSZ(ssz)
	if nil == err {
		s.beaconStateBellatrix = nil
		s.beaconStateAltair = nil
		s.beaconStatePhase0 = &phase0
		log.Info("Unmarshalled Phase0 BeaconState")
		return nil
	}

	s.beaconStateBellatrix = nil
	s.beaconStateAltair = nil
	s.beaconStatePhase0 = nil

	log.Warning("Unable to unmarshal BeaconState")
	return err
}

func (s *BeaconState) MarshalSSZ() ([]byte, error) {
	if s.IsBellatrix() {
		return s.beaconStateBellatrix.MarshalSSZ()
	}
	if s.IsAltair() {
		return s.beaconStateAltair.MarshalSSZ()
	}
	if s.IsPhase0() {
		return s.beaconStatePhase0.MarshalSSZ()
	}

	return []byte{}, errors.New("BeaconState not set")
}

func (s *BeaconState) IsBellatrix() bool {
	return s.beaconStateBellatrix != nil
}

func (s *BeaconState) IsAltair() bool {
	return s.beaconStateAltair != nil
}

func (s *BeaconState) IsPhase0() bool {
	return s.beaconStatePhase0 != nil
}

func (s *BeaconState) Slot() uint64 {
	if s.IsBellatrix() {
		return s.beaconStateBellatrix.Slot
	}

	if s.IsAltair() {
		return s.beaconStateAltair.Slot
	}

	if s.IsPhase0() {
		return s.beaconStatePhase0.Slot
	}

	// TODO(telackey): Something better than 0?
	return 0
}

func (b *BeaconState) HashTreeRoot() ([32]byte, error) {
	if b.IsBellatrix() {
		return b.beaconStateBellatrix.HashTreeRoot()
	}

	if b.IsAltair() {
		return b.beaconStateAltair.HashTreeRoot()
	}

	if b.IsPhase0() {
		return b.beaconStatePhase0.HashTreeRoot()
	}

	return [32]byte{}, errors.New("BeaconState not set")
}

func (s *BeaconState) GetBellatrix() *consensus.BeaconStateBellatrix {
	return s.beaconStateBellatrix
}

func (s *BeaconState) GetAltair() *consensus.BeaconStateAltair {
	return s.beaconStateAltair
}

func (s *BeaconState) GetPhase0() *consensus.BeaconStatePhase0 {
	return s.beaconStatePhase0
}
