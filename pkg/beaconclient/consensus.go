package beaconclient

import (
	"bytes"
	"errors"
	"github.com/protolambda/zrnt/eth2/beacon/altair"
	"github.com/protolambda/zrnt/eth2/beacon/bellatrix"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/beacon/phase0"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/tree"
	log "github.com/sirupsen/logrus"
)

type Eth1Data common.Eth1Data
type Root common.Root
type Signature common.BLSSignature
type Slot common.Slot

type BeaconBlock struct {
	spec      *common.Spec
	bellatrix *bellatrix.BeaconBlock
	altair    *altair.BeaconBlock
	phase0    *phase0.BeaconBlock
}

type BeaconBlockBody struct {
	spec      *common.Spec
	bellatrix *bellatrix.BeaconBlockBody
	altair    *altair.BeaconBlockBody
	phase0    *phase0.BeaconBlockBody
}

type BeaconState struct {
	spec      *common.Spec
	bellatrix *bellatrix.BeaconState
	altair    *altair.BeaconState
	phase0    *phase0.BeaconState
}

type SignedBeaconBlock struct {
	spec      *common.Spec
	bellatrix *bellatrix.SignedBeaconBlock
	altair    *altair.SignedBeaconBlock
	phase0    *phase0.SignedBeaconBlock
}

func (s *SignedBeaconBlock) UnmarshalSSZ(ssz []byte) error {
	spec := chooseSpec(s.spec)

	var bellatrix bellatrix.SignedBeaconBlock
	err := bellatrix.Deserialize(spec, makeDecodingReader(ssz))
	if nil == err {
		s.bellatrix = &bellatrix
		s.altair = nil
		s.phase0 = nil
		log.Info("Unmarshalled Bellatrix SignedBeaconBlock")
		return nil
	}

	var altair altair.SignedBeaconBlock
	err = altair.Deserialize(spec, makeDecodingReader(ssz))
	if nil == err {
		s.bellatrix = nil
		s.altair = &altair
		s.phase0 = nil
		log.Info("Unmarshalled Altair SignedBeaconBlock")
		return nil
	}

	var phase0 phase0.SignedBeaconBlock
	err = phase0.Deserialize(spec, makeDecodingReader(ssz))
	if nil == err {
		s.bellatrix = nil
		s.altair = nil
		s.phase0 = &phase0
		log.Info("Unmarshalled Phase0 SignedBeaconBlock")
		return nil
	}

	s.bellatrix = nil
	s.altair = nil
	s.phase0 = nil

	log.Warning("Unable to unmarshal SignedBeaconBlock")
	return err
}

func (s *SignedBeaconBlock) MarshalSSZ() ([]byte, error) {
	spec := chooseSpec(s.spec)
	var err error
	var buf bytes.Buffer
	encodingWriter := codec.NewEncodingWriter(&buf)

	if s.IsBellatrix() {
		err = s.bellatrix.Serialize(spec, encodingWriter)
	}
	if s.IsAltair() {
		err = s.altair.Serialize(spec, encodingWriter)
	}
	if s.IsPhase0() {
		err = s.phase0.Serialize(spec, encodingWriter)
	}

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), err
}

func (s *SignedBeaconBlock) IsBellatrix() bool {
	return s.bellatrix != nil
}

func (s *SignedBeaconBlock) IsAltair() bool {
	return s.altair != nil
}

func (s *SignedBeaconBlock) IsPhase0() bool {
	return s.phase0 != nil
}

func (s *SignedBeaconBlock) GetBellatrix() *bellatrix.SignedBeaconBlock {
	return s.bellatrix
}

func (s *SignedBeaconBlock) GetAltair() *altair.SignedBeaconBlock {
	return s.altair
}

func (s *SignedBeaconBlock) GetPhase0() *phase0.SignedBeaconBlock {
	return s.phase0
}

func (s *SignedBeaconBlock) Signature() Signature {
	if s.IsBellatrix() {
		return Signature(s.bellatrix.Signature)
	}

	if s.IsAltair() {
		return Signature(s.altair.Signature)
	}

	if s.IsPhase0() {
		return Signature(s.phase0.Signature)
	}

	return Signature{}
}

func (s *SignedBeaconBlock) Block() *BeaconBlock {
	if s.IsBellatrix() {
		return &BeaconBlock{bellatrix: &s.bellatrix.Message, spec: s.spec}
	}

	if s.IsAltair() {
		return &BeaconBlock{altair: &s.altair.Message, spec: s.spec}
	}

	if s.IsPhase0() {
		return &BeaconBlock{phase0: &s.phase0.Message, spec: s.spec}
	}

	return nil
}

func (b *BeaconBlock) IsBellatrix() bool {
	return b.bellatrix != nil
}

func (b *BeaconBlock) IsAltair() bool {
	return b.altair != nil
}

func (b *BeaconBlock) IsPhase0() bool {
	return b.phase0 != nil
}

func (s *BeaconBlock) GetBellatrix() *bellatrix.BeaconBlock {
	return s.bellatrix
}

func (s *BeaconBlock) GetAltair() *altair.BeaconBlock {
	return s.altair
}

func (s *BeaconBlock) GetPhase0() *phase0.BeaconBlock {
	return s.phase0
}

func (b *BeaconBlock) ParentRoot() Root {
	if b.IsBellatrix() {
		return Root(b.bellatrix.ParentRoot)
	}

	if b.IsAltair() {
		return Root(b.altair.ParentRoot)
	}

	if b.IsPhase0() {
		return Root(b.phase0.ParentRoot)
	}

	return Root{}
}

func (b *BeaconBlock) StateRoot() Root {
	if b.IsBellatrix() {
		return Root(b.bellatrix.StateRoot)
	}

	if b.IsAltair() {
		return Root(b.altair.StateRoot)
	}

	if b.IsPhase0() {
		return Root(b.phase0.StateRoot)
	}

	return Root{}
}

func (b *BeaconBlock) Body() *BeaconBlockBody {
	if b.IsBellatrix() {
		return &BeaconBlockBody{bellatrix: &b.bellatrix.Body, spec: b.spec}
	}

	if b.IsAltair() {
		return &BeaconBlockBody{altair: &b.altair.Body, spec: b.spec}
	}

	if b.IsPhase0() {
		return &BeaconBlockBody{phase0: &b.phase0.Body, spec: b.spec}
	}

	return nil
}

func (b *BeaconBlockBody) IsBellatrix() bool {
	return b.bellatrix != nil
}

func (b *BeaconBlockBody) IsAltair() bool {
	return b.altair != nil
}

func (b *BeaconBlockBody) IsPhase0() bool {
	return b.phase0 != nil
}

func (b *BeaconBlockBody) Eth1Data() Eth1Data {
	if b.IsBellatrix() {
		return Eth1Data(b.bellatrix.Eth1Data)
	}

	if b.IsAltair() {
		return Eth1Data(b.altair.Eth1Data)
	}

	if b.IsPhase0() {
		return Eth1Data(b.phase0.Eth1Data)
	}

	return Eth1Data{}
}

func (b *BeaconBlock) HashTreeRoot() Root {
	spec := chooseSpec(b.spec)
	hashFn := tree.GetHashFn()

	if b.IsBellatrix() {
		return Root(b.bellatrix.HashTreeRoot(spec, hashFn))
	}

	if b.IsAltair() {
		return Root(b.altair.HashTreeRoot(spec, hashFn))
	}

	if b.IsPhase0() {
		return Root(b.phase0.HashTreeRoot(spec, hashFn))
	}

	return Root{}
}

func (s *BeaconState) UnmarshalSSZ(ssz []byte) error {
	spec := chooseSpec(s.spec)

	var bellatrix bellatrix.BeaconState
	err := bellatrix.Deserialize(spec, makeDecodingReader(ssz))
	if nil == err {
		s.bellatrix = &bellatrix
		s.altair = nil
		s.phase0 = nil
		log.Info("Unmarshalled Bellatrix BeaconState")
		return nil
	}

	var altair altair.BeaconState
	err = altair.Deserialize(spec, makeDecodingReader(ssz))
	if nil == err {
		s.bellatrix = nil
		s.altair = &altair
		s.phase0 = nil
		log.Info("Unmarshalled Altair BeaconState")
		return nil
	}

	var phase0 phase0.BeaconState
	err = phase0.Deserialize(spec, makeDecodingReader(ssz))
	if nil == err {
		s.bellatrix = nil
		s.altair = nil
		s.phase0 = &phase0
		log.Info("Unmarshalled Phase0 BeaconState")
		return nil
	}

	s.bellatrix = nil
	s.altair = nil
	s.phase0 = nil

	log.Warning("Unable to unmarshal BeaconState")
	return err
}

func (s *BeaconState) MarshalSSZ() ([]byte, error) {
	spec := chooseSpec(s.spec)
	var err error
	var buf bytes.Buffer
	encodingWriter := codec.NewEncodingWriter(&buf)

	if s.IsBellatrix() {
		err = s.bellatrix.Serialize(spec, encodingWriter)
	} else if s.IsAltair() {
		err = s.altair.Serialize(spec, encodingWriter)
	} else if s.IsPhase0() {
		err = s.phase0.Serialize(spec, encodingWriter)
	} else {
		err = errors.New("BeaconState not set")
	}

	if nil != err {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *BeaconState) IsBellatrix() bool {
	return s.bellatrix != nil
}

func (s *BeaconState) IsAltair() bool {
	return s.altair != nil
}

func (s *BeaconState) IsPhase0() bool {
	return s.phase0 != nil
}

func (s *BeaconState) HashTreeRoot() Root {
	spec := chooseSpec(s.spec)
	hashFn := tree.GetHashFn()

	if s.IsBellatrix() {
		return Root(s.bellatrix.HashTreeRoot(spec, hashFn))
	}

	if s.IsAltair() {
		return Root(s.altair.HashTreeRoot(spec, hashFn))
	}

	if s.IsPhase0() {
		return Root(s.phase0.HashTreeRoot(spec, hashFn))
	}

	return Root{}
}

func (s *BeaconState) GetBellatrix() *bellatrix.BeaconState {
	return s.bellatrix
}

func (s *BeaconState) GetAltair() *altair.BeaconState {
	return s.altair
}

func (s *BeaconState) GetPhase0() *phase0.BeaconState {
	return s.phase0
}

func chooseSpec(spec *common.Spec) *common.Spec {
	if nil == spec {
		return configs.Mainnet
	}
	return spec
}

func makeDecodingReader(ssz []byte) *codec.DecodingReader {
	return codec.NewDecodingReader(bytes.NewReader(ssz), uint64(len(ssz)))
}
