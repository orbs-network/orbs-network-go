package statestorage

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/statestorage/merkle"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"sync"
)

var LogTag = log.Service("state-storage")

type service struct {
	config       config.StateStorageConfig
	blockTracker *synchronization.BlockTracker
	logger       log.BasicLogger

	mutex     sync.RWMutex
	revisions *rollingRevisions
}

func NewStateStorage(config config.StateStorageConfig, persistence adapter.StatePersistence, logger log.BasicLogger) services.StateStorage {

	forest, _ := merkle.NewForest()
	return &service{
		config:       config,
		blockTracker: synchronization.NewBlockTracker(0, uint16(config.BlockTrackerGraceDistance())),
		logger:       logger.WithTags(LogTag),

		mutex:     sync.RWMutex{},
		revisions: newRollingRevisions(persistence, int(config.StateStorageHistoryRetentionDistance()), forest),
	}
}

func (s *service) CommitStateDiff(ctx context.Context, input *services.CommitStateDiffInput) (*services.CommitStateDiffOutput, error) {
	if input.ResultsBlockHeader == nil || input.ContractStateDiffs == nil {
		panic("CommitStateDiff received corrupt args")
	}

	commitBlockHeight := input.ResultsBlockHeader.BlockHeight()
	commitTimestamp := input.ResultsBlockHeader.Timestamp()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.logger.Info("trying to commit state diff", log.BlockHeight(commitBlockHeight), log.Int("number-of-state-diffs", len(input.ContractStateDiffs)))

	currentHeight := s.revisions.getCurrentHeight()
	if currentHeight+1 != commitBlockHeight {
		return &services.CommitStateDiffOutput{NextDesiredBlockHeight: currentHeight + 1}, nil
	}

	// TODO assert input.ResultsBlockHeader.PreExecutionStateRootHash() == s.revisions.getRevisionHash(commitBlockHeight - 1)

	err := s.revisions.addRevision(commitBlockHeight, commitTimestamp, inflateChainState(input.ContractStateDiffs))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to write state for block height %d", commitBlockHeight)
	}

	s.blockTracker.IncrementHeight()
	return &services.CommitStateDiffOutput{NextDesiredBlockHeight: commitBlockHeight + 1}, nil
}

func (s *service) ReadKeys(ctx context.Context, input *services.ReadKeysInput) (*services.ReadKeysOutput, error) {
	if input.ContractName == "" {
		return nil, errors.Errorf("missing contract name")
	}

	timeoutCtx, _ := context.WithTimeout(ctx, s.config.BlockTrackerGraceTimeout())
	if err := s.blockTracker.WaitForBlock(timeoutCtx, input.BlockHeight); err != nil {
		return nil, errors.Wrapf(err, "unsupported block height: block %v is not yet committed", input.BlockHeight)
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	currentHeight := s.revisions.getCurrentHeight()
	if input.BlockHeight+primitives.BlockHeight(s.config.StateStorageHistoryRetentionDistance()) <= currentHeight {
		return nil, errors.Errorf("unsupported block height: block %v too old. currently at %v. keeping %v back", input.BlockHeight, currentHeight, primitives.BlockHeight(s.config.StateStorageHistoryRetentionDistance()))
	}

	records := make([]*protocol.StateRecord, 0, len(input.Keys))
	for _, key := range input.Keys {
		record, ok, err := s.revisions.getRevisionRecord(input.BlockHeight, input.ContractName, key.KeyForMap())
		if err != nil {
			return nil, errors.Wrap(err, "persistence layer error")
		}
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

func (s *service) GetStateStorageBlockHeight(ctx context.Context, input *services.GetStateStorageBlockHeightInput) (*services.GetStateStorageBlockHeightOutput, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := &services.GetStateStorageBlockHeightOutput{
		LastCommittedBlockHeight:    s.revisions.getCurrentHeight(),
		LastCommittedBlockTimestamp: s.revisions.getCurrentTimestamp(),
	}
	return result, nil
}

func (s *service) GetStateHash(ctx context.Context, input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {
	timeoutCtx, _ := context.WithTimeout(ctx, s.config.BlockTrackerGraceTimeout())
	if err := s.blockTracker.WaitForBlock(timeoutCtx, input.BlockHeight); err != nil {
		return nil, errors.Wrapf(err, "unsupported block height: block %v is not yet committed", input.BlockHeight)
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	currentHeight := s.revisions.getCurrentHeight()
	if input.BlockHeight+primitives.BlockHeight(s.config.StateStorageHistoryRetentionDistance()) <= currentHeight {
		return nil, errors.Errorf("unsupported block height: block %v too old. currently at %v. keeping %v back", input.BlockHeight, currentHeight, primitives.BlockHeight(s.config.StateStorageHistoryRetentionDistance()))
	}

	value, err := s.revisions.getRevisionHash(input.BlockHeight)
	if err != nil {
		return nil, errors.Wrapf(err, "could not find a merkle root for block height %d", input.BlockHeight)
	}
	output := &services.GetStateHashOutput{StateRootHash: value}

	return output, nil
}

func inflateChainState(csd []*protocol.ContractStateDiff) adapter.ChainState {
	result := make(adapter.ChainState)
	for _, stateDiffs := range csd {
		contract := stateDiffs.ContractName()
		contractMap, ok := result[contract]
		if !ok {
			contractMap = make(map[string]*protocol.StateRecord)
			result[contract] = contractMap
		}
		for i := stateDiffs.StateDiffsIterator(); i.HasNext(); {
			r := i.NextStateDiffs()
			contractMap[r.Key().KeyForMap()] = r
		}
	}
	return result
}

func newZeroValue() []byte {
	return []byte{}
}
