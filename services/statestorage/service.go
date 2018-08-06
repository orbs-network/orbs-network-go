package statestorage

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/statestorage/merkle"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

type Config interface {
	GetMaxStateHistory() uint64
}

type service struct {
	persistence            adapter.StatePersistence
	merkle                 *merkle.Forest
	lastResultsBlockHeader *protocol.ResultsBlockHeader
	config                 Config
}

func NewStateStorage(config Config, persistence adapter.StatePersistence) services.StateStorage {
	return &service{
		persistence:            persistence,
		merkle:                 merkle.NewForest(),
		lastResultsBlockHeader: (&protocol.ResultsBlockHeaderBuilder{}).Build(), // TODO change when system inits genesis block and saves it
		config:                 config,
	}
}

func (s *service) CommitStateDiff(input *services.CommitStateDiffInput) (*services.CommitStateDiffOutput, error) {
	committedBlock := input.ResultsBlockHeader.BlockHeight()
	if lastCommittedBlock := s.lastResultsBlockHeader.BlockHeight(); lastCommittedBlock+1 != committedBlock {
		return &services.CommitStateDiffOutput{NextDesiredBlockHeight: lastCommittedBlock + 1}, nil
	}

	// if updating state records fails downstream the merkle tree entries will not bother us
	s.merkle.Update(merkle.RootId(committedBlock), input.ContractStateDiffs)

	//TODO how do we handle partial failures?
	for _, stateDiffs := range input.ContractStateDiffs {
		for i := stateDiffs.StateDiffsIterator(); i.HasNext(); {
			s.persistence.WriteState(committedBlock, stateDiffs.ContractName(), i.NextStateDiffs())
		}
	}

	s.lastResultsBlockHeader = input.ResultsBlockHeader
	height := committedBlock + 1
	return &services.CommitStateDiffOutput{NextDesiredBlockHeight: height}, nil
}

func (s *service) ReadKeys(input *services.ReadKeysInput) (*services.ReadKeysOutput, error) {
	if input.ContractName == "" {
		return nil, fmt.Errorf("missing contract name")
	}

	if input.BlockHeight+primitives.BlockHeight(s.config.GetMaxStateHistory()) <= s.lastResultsBlockHeader.BlockHeight() {
		return nil, fmt.Errorf("unsupported block height: block %v too old. currently at %v. keeping %v back", input.BlockHeight, s.lastResultsBlockHeader.BlockHeight(), primitives.BlockHeight(s.config.GetMaxStateHistory()))
	}

	contractState, err := s.persistence.ReadState(input.BlockHeight, input.ContractName)
	if err != nil {
		return nil, errors.Wrap(err, "persistence layer error")
	}

	records := make([]*protocol.StateRecord, 0, len(input.Keys))
	for _, key := range input.Keys {
		record, ok := contractState[key.KeyForMap()]
		if ok {
			records = append(records, record)
		} else { // implicitly return the zero value if key is missing in db
			records = append(records, (&protocol.StateRecordBuilder{Key: key, Value: []byte{}}).Build())
		}
	}

	output := &services.ReadKeysOutput{StateRecords: records}
	if len(output.StateRecords) == 0 {
		return output, fmt.Errorf("no value found for input key(s)")
	}
	return output, nil
}

func (s *service) GetStateStorageBlockHeight(input *services.GetStateStorageBlockHeightInput) (*services.GetStateStorageBlockHeightOutput, error) {
	result := &services.GetStateStorageBlockHeightOutput{
		LastCommittedBlockHeight:    s.lastResultsBlockHeader.BlockHeight(),
		LastCommittedBlockTimestamp: s.lastResultsBlockHeader.Timestamp(),
	}
	return result, nil
}

func (s *service) GetStateHash(input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {
	value, _ := s.merkle.GetRoot(merkle.RootId(input.BlockHeight))

	output := &services.GetStateHashOutput{StateRootHash: value}

	return output, nil
}
