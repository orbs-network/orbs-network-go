package statestorage

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/statestorage/merkle"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"sync"
	"time"
)

type Config interface {
	StateHistoryRetentionInBlockHeights() uint64
	QuerySyncGraceBlockDist() uint64
	QueryGraceTimeoutMillis() uint64
}

type service struct {
	config Config
	merkle *merkle.Forest

	mutex                    *sync.RWMutex
	persistence              adapter.StatePersistence
	lastCommittedBlockHeader *protocol.ResultsBlockHeader

	blockTracker *BlockTracker
}

func NewStateStorage(config Config, persistence adapter.StatePersistence) services.StateStorage {
	return &service{
		config:       config,
		merkle:       merkle.NewForest(),
		blockTracker: NewBlockTracker(0, uint16(config.QuerySyncGraceBlockDist()), time.Duration(config.QueryGraceTimeoutMillis())*time.Millisecond),

		mutex:                    &sync.RWMutex{},
		persistence:              persistence,
		lastCommittedBlockHeader: (&protocol.ResultsBlockHeaderBuilder{}).Build(), // TODO change when system inits genesis block and saves it
	}
}

func (s *service) CommitStateDiff(input *services.CommitStateDiffInput) (*services.CommitStateDiffOutput, error) {
	if input.ResultsBlockHeader == nil || input.ContractStateDiffs == nil {
		panic("CommitStateDiff received corrupt args")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	committedBlock := input.ResultsBlockHeader.BlockHeight()
	fmt.Printf("trying to commit state diff for block height %d, num contract state diffs %d\n", committedBlock, len(input.ContractStateDiffs)) // TODO: move this to reporting mechanism

	if lastCommittedBlock := s.lastCommittedBlockHeader.BlockHeight(); lastCommittedBlock+1 != committedBlock {
		return &services.CommitStateDiffOutput{NextDesiredBlockHeight: lastCommittedBlock + 1}, nil
	}

	// if updating state records fails downstream the merkle tree entries will not bother us
	s.merkle.Update(merkle.RootId(committedBlock), input.ContractStateDiffs)
	s.persistence.WriteState(committedBlock, input.ContractStateDiffs)

	s.lastCommittedBlockHeader = input.ResultsBlockHeader
	height := committedBlock + 1

	s.blockTracker.IncrementHeight()

	return &services.CommitStateDiffOutput{NextDesiredBlockHeight: height}, nil
}

func (s *service) ReadKeys(input *services.ReadKeysInput) (*services.ReadKeysOutput, error) {
	if input.ContractName == "" {
		return nil, fmt.Errorf("missing contract name")
	}

	if input.BlockHeight+primitives.BlockHeight(s.config.StateHistoryRetentionInBlockHeights()) <= s.lastCommittedBlockHeader.BlockHeight() {
		return nil, fmt.Errorf("unsupported block height: block %v too old. currently at %v. keeping %v back", input.BlockHeight, s.lastCommittedBlockHeader.BlockHeight(), primitives.BlockHeight(s.config.StateHistoryRetentionInBlockHeights()))
	}

	if err := s.blockTracker.WaitForBlock(input.BlockHeight); err != nil {
		return nil, errors.Wrapf(err, "unsupported block height: block %v is not yet committed", input.BlockHeight)
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

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
		LastCommittedBlockHeight:    s.lastCommittedBlockHeader.BlockHeight(),
		LastCommittedBlockTimestamp: s.lastCommittedBlockHeader.Timestamp(),
	}
	return result, nil
}

func (s *service) GetStateHash(input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {
	value, _ := s.merkle.GetRoot(merkle.RootId(input.BlockHeight))

	output := &services.GetStateHashOutput{StateRootHash: value}

	return output, nil
}
