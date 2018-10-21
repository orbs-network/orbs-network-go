package statestorage

import (
	"bytes"
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

type stateIncrement struct {
	diff       adapter.ChainDiff
	merkleRoot primitives.MerkleSha256
	height     primitives.BlockHeight
	ts         primitives.TimestampNano
}

type service struct {
	config       config.StateStorageConfig
	blockTracker *synchronization.BlockTracker
	logger       log.BasicLogger

	mutex       sync.RWMutex
	incCache    []stateIncrement
	persistence adapter.StatePersistence
	merkle      *merkle.Forest
	height      primitives.BlockHeight
	ts          primitives.TimestampNano
}

func NewStateStorage(config config.StateStorageConfig, persistence adapter.StatePersistence, logger log.BasicLogger) services.StateStorage {
	// TODO - tie/sync merkle forest to persistent state
	merkle, _ := merkle.NewForest()

	height, err := persistence.ReadBlockHeight()
	if err != nil {
		panic(err)
	}
	ts, err := persistence.ReadBlockTimestamp()
	if err != nil {
		panic(err)
	}

	return &service{
		config:       config,
		blockTracker: synchronization.NewBlockTracker(0, uint16(config.BlockTrackerGraceDistance()), config.BlockTrackerGraceTimeout()),
		logger:       logger.WithTags(LogTag),

		mutex:       sync.RWMutex{},
		merkle:      merkle,
		incCache:    []stateIncrement{},
		persistence: persistence,
		height:      height,
		ts:          ts,
	}
}

func (s *service) CommitStateDiff(ctx context.Context, input *services.CommitStateDiffInput) (*services.CommitStateDiffOutput, error) {
	if input.ResultsBlockHeader == nil || input.ContractStateDiffs == nil {
		panic("CommitStateDiff received corrupt args")
	}

	commitBlockHeight := input.ResultsBlockHeader.BlockHeight()
	commitTimestamp := input.ResultsBlockHeader.Timestamp()
	persistedBlockHeight, err := s.persistence.ReadBlockHeight()

	if err != nil {
		return nil, errors.Wrapf(err, "failed to read persisted block height")
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.logger.Info("trying to commit state diff", log.BlockHeight(commitBlockHeight), log.Int("number-of-state-diffs", len(input.ContractStateDiffs)))

	if lastCommittedBlock := s.height; lastCommittedBlock+1 != commitBlockHeight {
		return &services.CommitStateDiffOutput{NextDesiredBlockHeight: lastCommittedBlock + 1}, nil
	}

	// if updating state records fails downstream the merkle tree entries will not bother us
	// TODO use input.resultheader.preexecutuion
	root, err := s._readStateHash(commitBlockHeight - 1)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot find previous block merkle root. current block %d", commitBlockHeight)
	}
	newRoot, err := s.merkle.Update(root, input.ContractStateDiffs)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot find previous block merkle root. current block %d", commitBlockHeight)
	}

	if commitBlockHeight > persistedBlockHeight+primitives.BlockHeight(s.config.StateStorageHistoryRetentionDistance()) {
		d := s.incCache[0]
		s.persistence.WriteState(d.height, d.ts, d.merkleRoot, d.diff)
		s.incCache = s.incCache[1:]
	}
	s.incCache = append(s.incCache, stateIncrement{
		diff:       _newChainDiff(input.ContractStateDiffs),
		merkleRoot: newRoot,
		height:     commitBlockHeight,
		ts:         commitTimestamp,
	})
	s.height = commitBlockHeight
	s.ts = commitTimestamp
	s.blockTracker.IncrementHeight()

	return &services.CommitStateDiffOutput{NextDesiredBlockHeight: commitBlockHeight + 1}, nil
}

func _newChainDiff(csd []*protocol.ContractStateDiff) adapter.ChainDiff {
	result := make(adapter.ChainDiff)
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

func (s *service) ReadKeys(ctx context.Context, input *services.ReadKeysInput) (*services.ReadKeysOutput, error) {
	if input.ContractName == "" {
		return nil, errors.Errorf("missing contract name")
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if input.BlockHeight+primitives.BlockHeight(s.config.StateStorageHistoryRetentionDistance()) <= s.height {
		return nil, errors.Errorf("unsupported block height: block %v too old. currently at %v. keeping %v back", input.BlockHeight, s.height, primitives.BlockHeight(s.config.StateStorageHistoryRetentionDistance()))
	}

	if err := s.blockTracker.WaitForBlock(ctx, input.BlockHeight); err != nil {
		return nil, errors.Wrapf(err, "unsupported block height: block %v is not yet committed", input.BlockHeight)
	}

	if input.BlockHeight+primitives.BlockHeight(s.config.StateStorageHistoryRetentionDistance()) <= s.height {
		return nil, errors.Errorf("unsupported block height: block %v too old. currently at %v. keeping %v back", input.BlockHeight, s.height, primitives.BlockHeight(s.config.StateStorageHistoryRetentionDistance()))
	}

	records := make([]*protocol.StateRecord, 0, len(input.Keys))
	for _, key := range input.Keys {
		record, ok, err := s._readStateKey(input.BlockHeight, input.ContractName, key.KeyForMap())
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

func (s *service) _readStateKey(height primitives.BlockHeight, contract primitives.ContractName, key string) (*protocol.StateRecord, bool, error) {
	persistedHeight, err := s.persistence.ReadBlockHeight()

	if err != nil {
		return nil, false, errors.Wrap(err, "could not find base block height")
	}

	cacheIdx := int(height - persistedHeight - 1)
	if cacheIdx >= len(s.incCache) {
		return nil, false, errors.Errorf("accessing block state diff that has not been received yet")
	}

	for i := cacheIdx; i >= 0; i-- {
		if record, exists := s.incCache[i].diff[contract][key]; exists {
			return record, !isZeroValue(record.Value()), nil // cached state increments must include zero values
		}
	}
	return s.persistence.ReadState(persistedHeight, contract, key)
}

func (s *service) _readStateHash(height primitives.BlockHeight) (primitives.MerkleSha256, error) {
	persistedHeight, err := s.persistence.ReadBlockHeight()

	if err != nil {
		return nil, errors.Wrap(err, "could not find base block height")
	}

	if height == persistedHeight {
		return s.persistence.ReadMerkleRoot(persistedHeight)
	}

	cacheIdx := height - persistedHeight - 1
	if cacheIdx >= primitives.BlockHeight(len(s.incCache)) {
		return nil, errors.Errorf("accessing block state diff that has not been received yet")
	}

	return s.incCache[cacheIdx].merkleRoot, nil
}

func (s *service) GetStateStorageBlockHeight(ctx context.Context, input *services.GetStateStorageBlockHeightInput) (*services.GetStateStorageBlockHeightOutput, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := &services.GetStateStorageBlockHeightOutput{
		LastCommittedBlockHeight:    s.height,
		LastCommittedBlockTimestamp: s.ts,
	}
	return result, nil
}

func (s *service) GetStateHash(ctx context.Context, input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {
	if err := s.blockTracker.WaitForBlock(ctx, input.BlockHeight); err != nil {
		return nil, errors.Wrapf(err, "unsupported block height: block %v is not yet committed", input.BlockHeight)
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	value, err := s._readStateHash(input.BlockHeight)
	if err != nil {
		return nil, errors.Wrapf(err, "could not find a merkle root for block height %d", input.BlockHeight)
	}
	output := &services.GetStateHashOutput{StateRootHash: value}

	return output, nil
}

func newZeroValue() []byte {
	return []byte{}
}

func isZeroValue(value []byte) bool {
	return bytes.Equal(value, []byte{})
}
