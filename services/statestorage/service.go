package statestorage

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/statestorage/merkle"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"sync"
	"time"
)

type Config interface {
	StateHistoryRetentionInBlockHeights() uint16
	QuerySyncGraceBlockDist() uint16
	QueryGraceTimeoutMillis() uint64
}

type service struct {
	config       Config
	merkle       *merkle.Forest
	blockTracker *synchronization.BlockTracker
	reporting    log.BasicLogger

	mutex                    *sync.RWMutex
	persistence              adapter.StatePersistence
	lastCommittedBlockHeader *protocol.ResultsBlockHeader
}

func NewStateStorage(config Config, persistence adapter.StatePersistence, reporting log.BasicLogger) services.StateStorage {
	return &service{
		config:       config,
		merkle:       merkle.NewForest(),
		blockTracker: synchronization.NewBlockTracker(0, uint16(config.QuerySyncGraceBlockDist()), time.Duration(config.QueryGraceTimeoutMillis())*time.Millisecond),
		reporting:    reporting.For(log.Service("state-storage")),

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

	commitBlockHeight := input.ResultsBlockHeader.BlockHeight()

	s.reporting.Info("trying to commit state diff", log.BlockHeight(commitBlockHeight), log.Int("number-of-state-diffs", len(input.ContractStateDiffs)))

	if lastCommittedBlock := s.lastCommittedBlockHeader.BlockHeight(); lastCommittedBlock+1 != commitBlockHeight {
		return &services.CommitStateDiffOutput{NextDesiredBlockHeight: lastCommittedBlock + 1}, nil
	}

	// if updating state records fails downstream the merkle tree entries will not bother us
	s.merkle.Update(input.ContractStateDiffs)
	s.persistence.WriteState(commitBlockHeight, input.ContractStateDiffs)

	s.lastCommittedBlockHeader = input.ResultsBlockHeader
	s.blockTracker.IncrementHeight()

	return &services.CommitStateDiffOutput{NextDesiredBlockHeight: commitBlockHeight + 1}, nil
}

func (s *service) ReadKeys(input *services.ReadKeysInput) (*services.ReadKeysOutput, error) {
	if input.ContractName == "" {
		return nil, errors.Errorf("missing contract name")
	}

	if err := s.blockTracker.WaitForBlock(input.BlockHeight); err != nil {
		return nil, errors.Wrapf(err, "unsupported block height: block %v is not yet committed", input.BlockHeight)
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if input.BlockHeight+primitives.BlockHeight(s.config.StateHistoryRetentionInBlockHeights()) <= s.lastCommittedBlockHeader.BlockHeight() {
		return nil, errors.Errorf("unsupported block height: block %v too old. currently at %v. keeping %v back", input.BlockHeight, s.lastCommittedBlockHeader.BlockHeight(), primitives.BlockHeight(s.config.StateHistoryRetentionInBlockHeights()))
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
		return output, errors.Errorf("no value found for input key(s)")
	}
	return output, nil
}

func (s *service) GetStateStorageBlockHeight(input *services.GetStateStorageBlockHeightInput) (*services.GetStateStorageBlockHeightOutput, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := &services.GetStateStorageBlockHeightOutput{
		LastCommittedBlockHeight:    s.lastCommittedBlockHeader.BlockHeight(),
		LastCommittedBlockTimestamp: s.lastCommittedBlockHeader.Timestamp(),
	}
	return result, nil
}

func (s *service) GetStateHash(input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {

	if err := s.blockTracker.WaitForBlock(input.BlockHeight); err != nil {
		return nil, errors.Wrapf(err, "unsupported block height: block %v is not yet committed", input.BlockHeight)
	}

	value, _ := s.merkle.GetRootHash(merkle.TrieId(input.BlockHeight))

	output := &services.GetStateHashOutput{StateRootHash: value}

	return output, nil
}
