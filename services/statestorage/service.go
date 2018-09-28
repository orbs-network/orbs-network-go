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

var LogTag = log.Service("state-storage")

type Config interface {
	StateStorageHistoryRetentionDistance() uint32
	BlockTrackerGraceDistance() uint32
	BlockTrackerGraceTimeout() time.Duration
}

type service struct {
	config       Config
	merkle       *merkle.Forest
	blockTracker *synchronization.BlockTracker
	logger       log.BasicLogger

	mutex                    *sync.RWMutex
	persistence              adapter.StatePersistence
	lastCommittedBlockHeader *protocol.ResultsBlockHeader
}

func NewStateStorage(config Config, persistence adapter.StatePersistence, logger log.BasicLogger) services.StateStorage {
	merkle, rootHash := merkle.NewForest()
	// TODO this is equivalent of genesis block deploy in persistence -> move to correct deploy
	persistence.WriteMerkleRoot(0, rootHash)

	return &service{
		config:       config,
		merkle:       merkle,
		blockTracker: synchronization.NewBlockTracker(0, uint16(config.BlockTrackerGraceDistance()), config.BlockTrackerGraceTimeout()),
		logger:       logger.WithTag(LogTag),

		mutex:                    &sync.RWMutex{},
		persistence:              persistence,
		lastCommittedBlockHeader: (&protocol.ResultsBlockHeaderBuilder{}).Build(), // TODO change when system inits genesis block and saves it will need to be read from db
	}
}

func (s *service) CommitStateDiff(input *services.CommitStateDiffInput) (*services.CommitStateDiffOutput, error) {
	if input.ResultsBlockHeader == nil || input.ContractStateDiffs == nil {
		panic("CommitStateDiff received corrupt args")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	commitBlockHeight := input.ResultsBlockHeader.BlockHeight()

	s.logger.Info("trying to commit state diff", log.BlockHeight(commitBlockHeight), log.Int("number-of-state-diffs", len(input.ContractStateDiffs)))

	if lastCommittedBlock := s.lastCommittedBlockHeader.BlockHeight(); lastCommittedBlock+1 != commitBlockHeight {
		return &services.CommitStateDiffOutput{NextDesiredBlockHeight: lastCommittedBlock + 1}, nil
	}

	// if updating state records fails downstream the merkle tree entries will not bother us
	// TODO use input.resultheader.preexecutuion
	root, err := s.persistence.ReadMerkleRoot(commitBlockHeight - 1)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot find previous block merkle root. current block %d", commitBlockHeight)
	}
	newRoot, err := s.merkle.Update(root, input.ContractStateDiffs)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot find previous block merkle root. current block %d", commitBlockHeight)
	}
	s.persistence.WriteMerkleRoot(commitBlockHeight, newRoot)
	s.persistence.WriteState(commitBlockHeight, input.ContractStateDiffs)

	s.lastCommittedBlockHeader = input.ResultsBlockHeader
	s.blockTracker.IncrementHeight()

	return &services.CommitStateDiffOutput{NextDesiredBlockHeight: commitBlockHeight + 1}, nil
}

func (s *service) ReadKeys(input *services.ReadKeysInput) (*services.ReadKeysOutput, error) {
	if input.ContractName == "" {
		return nil, errors.Errorf("missing contract name")
	}

	if input.BlockHeight+primitives.BlockHeight(s.config.StateStorageHistoryRetentionDistance()) <= s.lastCommittedBlockHeader.BlockHeight() {
		return nil, errors.Errorf("unsupported block height: block %v too old. currently at %v. keeping %v back", input.BlockHeight, s.lastCommittedBlockHeader.BlockHeight(), primitives.BlockHeight(s.config.StateStorageHistoryRetentionDistance()))
	}

	if err := s.blockTracker.WaitForBlock(input.BlockHeight); err != nil {
		return nil, errors.Wrapf(err, "unsupported block height: block %v is not yet committed", input.BlockHeight)
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if input.BlockHeight+primitives.BlockHeight(s.config.StateStorageHistoryRetentionDistance()) <= s.lastCommittedBlockHeader.BlockHeight() {
		return nil, errors.Errorf("unsupported block height: block %v too old. currently at %v. keeping %v back", input.BlockHeight, s.lastCommittedBlockHeader.BlockHeight(), primitives.BlockHeight(s.config.StateStorageHistoryRetentionDistance()))
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
			records = append(records, (&protocol.StateRecordBuilder{Key: key, Value: newZeroValue()}).Build())
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

	value, err := s.persistence.ReadMerkleRoot(input.BlockHeight)
	if err != nil {
		return nil, errors.Wrapf(err, "could not merkle root for block height %d", input.BlockHeight)
	}
	output := &services.GetStateHashOutput{StateRootHash: value}

	return output, nil
}

func newZeroValue() []byte {
	return []byte{}
}
