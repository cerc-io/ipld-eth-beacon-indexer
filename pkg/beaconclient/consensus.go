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

type Slot common.Slot
type Root common.Root

type Eth1Data common.Eth1Data

type SignedBeaconBlock struct {
	bellatrix *bellatrix.SignedBeaconBlock
	altair    *altair.SignedBeaconBlock
	phase0    *phase0.SignedBeaconBlock
}

type BeaconBlock struct {
	bellatrix *bellatrix.BeaconBlock
	altair    *altair.BeaconBlock
	phase0    *phase0.BeaconBlock
}

type BeaconBlockBody struct {
	bellatrix *bellatrix.BeaconBlockBody
	altair    *altair.BeaconBlockBody
	phase0    *phase0.BeaconBlockBody
}

type BeaconState struct {
	bellatrix *bellatrix.BeaconState
	altair    *altair.BeaconState
	phase0    *phase0.BeaconState
}

func (s *SignedBeaconBlock) UnmarshalSSZ(ssz []byte) error {
	var bellatrix bellatrix.SignedBeaconBlock
	decodingReader := codec.NewDecodingReader(bytes.NewReader(ssz), uint64(len(ssz)))
	err := bellatrix.Deserialize(configs.Mainnet, decodingReader)
	if nil == err {
		s.bellatrix = &bellatrix
		s.altair = nil
		s.phase0 = nil
		log.Info("Unmarshalled Bellatrix SignedBeaconBlock")
		return nil
	}

	var altair altair.SignedBeaconBlock
	decodingReader = codec.NewDecodingReader(bytes.NewReader(ssz), uint64(len(ssz)))
	err = altair.Deserialize(configs.Mainnet, decodingReader)
	if nil == err {
		s.bellatrix = nil
		s.altair = &altair
		s.phase0 = nil
		log.Info("Unmarshalled Altair SignedBeaconBlock")
		return nil
	}

	var phase0 phase0.SignedBeaconBlock
	decodingReader = codec.NewDecodingReader(bytes.NewReader(ssz), uint64(len(ssz)))
	err = phase0.Deserialize(configs.Mainnet, decodingReader)
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
	var err error
	var buf bytes.Buffer
	encodingWriter := codec.NewEncodingWriter(&buf)

	if s.IsBellatrix() {
		err = s.bellatrix.Serialize(configs.Mainnet, encodingWriter)
	}
	if s.IsAltair() {
		err = s.altair.Serialize(configs.Mainnet, encodingWriter)
	}
	if s.IsPhase0() {
		err = s.phase0.Serialize(configs.Mainnet, encodingWriter)
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

func (s *SignedBeaconBlock) Signature() [96]byte {
	if s.IsBellatrix() {
		return s.bellatrix.Signature
	}

	if s.IsAltair() {
		return s.altair.Signature
	}

	if s.IsPhase0() {
		return s.phase0.Signature
	}

	return [96]byte{}
}

func (s *SignedBeaconBlock) Block() *BeaconBlock {
	if s.IsBellatrix() {
		return &BeaconBlock{bellatrix: &s.bellatrix.Message}
	}

	if s.IsAltair() {
		return &BeaconBlock{altair: &s.altair.Message}
	}

	if s.IsPhase0() {
		return &BeaconBlock{phase0: &s.phase0.Message}
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
		return &BeaconBlockBody{bellatrix: &b.bellatrix.Body}
	}

	if b.IsAltair() {
		return &BeaconBlockBody{altair: &b.altair.Body}
	}

	if b.IsPhase0() {
		return &BeaconBlockBody{phase0: &b.phase0.Body}
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
	if b.IsBellatrix() {
		return Root(b.bellatrix.HashTreeRoot(configs.Mainnet, tree.Hash))
	}

	if b.IsAltair() {
		return Root(b.altair.HashTreeRoot(configs.Mainnet, tree.Hash))
	}

	if b.IsPhase0() {
		return Root(b.phase0.HashTreeRoot(configs.Mainnet, tree.Hash))
	}

	return Root{}
}

func (s *BeaconState) UnmarshalSSZ(ssz []byte) error {
	var bellatrix bellatrix.BeaconState
	decodingReader := codec.NewDecodingReader(bytes.NewReader(ssz), uint64(len(ssz)))
	err := bellatrix.Deserialize(configs.Mainnet, decodingReader)
	if nil == err {
		s.bellatrix = &bellatrix
		s.altair = nil
		s.phase0 = nil
		log.Info("Unmarshalled Bellatrix BeaconState")
		return nil
	}

	var altair altair.BeaconState
	decodingReader = codec.NewDecodingReader(bytes.NewReader(ssz), uint64(len(ssz)))
	err = altair.Deserialize(configs.Mainnet, decodingReader)
	if nil == err {
		s.bellatrix = nil
		s.altair = &altair
		s.phase0 = nil
		log.Info("Unmarshalled Altair BeaconState")
		return nil
	}

	var phase0 phase0.BeaconState
	decodingReader = codec.NewDecodingReader(bytes.NewReader(ssz), uint64(len(ssz)))
	err = phase0.Deserialize(configs.Mainnet, decodingReader)
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
	var err error
	var buf bytes.Buffer
	encodingWriter := codec.NewEncodingWriter(&buf)

	if s.IsBellatrix() {
		err = s.bellatrix.Serialize(configs.Mainnet, encodingWriter)
	} else if s.IsAltair() {
		err = s.altair.Serialize(configs.Mainnet, encodingWriter)
	} else if s.IsPhase0() {
		err = s.phase0.Serialize(configs.Mainnet, encodingWriter)
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

func (s *BeaconState) Slot() Slot {
	if s.IsBellatrix() {
		return Slot(s.bellatrix.Slot)
	}

	if s.IsAltair() {
		return Slot(s.altair.Slot)
	}

	if s.IsPhase0() {
		return Slot(s.phase0.Slot)
	}

	// TODO(telackey): Something better than 0?
	return 0
}

func (b *BeaconState) HashTreeRoot() Root {
	if b.IsBellatrix() {
		return Root(b.bellatrix.HashTreeRoot(configs.Mainnet, tree.Hash))
	}

	if b.IsAltair() {
		return Root(b.altair.HashTreeRoot(configs.Mainnet, tree.Hash))
	}

	if b.IsPhase0() {
		return Root(b.phase0.HashTreeRoot(configs.Mainnet, tree.Hash))
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
