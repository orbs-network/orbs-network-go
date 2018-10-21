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
	merkle, root := merkle.NewForest()

	height, ts, pRoot, err := persistence.ReadMetadata()
	if err != nil {
		panic(err)
	}
	if !pRoot.Equal(root) {
		panic("Merkle forest out of sync with persisted state")
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

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.logger.Info("trying to commit state diff", log.BlockHeight(commitBlockHeight), log.Int("number-of-state-diffs", len(input.ContractStateDiffs)))

	if s.height + 1 != commitBlockHeight {
		return &services.CommitStateDiffOutput{NextDesiredBlockHeight: s.height + 1}, nil
	}

	// if updating state records fails downstream the merkle tree entries will not bother us
	// TODO use input.resultheader.preexecutuion
	root, err := s._readStateHash(commitBlockHeight - 1)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot find previous block merkle root. current block %d", s.height)
	}
	newRoot, err := s.merkle.Update(root, input.ContractStateDiffs)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot find new merkle root. current block %d", s.height)
	}

	err = s._writeState(commitBlockHeight, commitTimestamp, newRoot, input)
	if err !=  nil {
		return nil, errors.Wrapf(err, "failed to write state for block height %d", commitBlockHeight)
	}

	s.height = commitBlockHeight
	s.ts = commitTimestamp
	s.blockTracker.IncrementHeight()
	return &services.CommitStateDiffOutput{NextDesiredBlockHeight: commitBlockHeight + 1}, nil
}

func (s *service) _writeState(height primitives.BlockHeight, ts primitives.TimestampNano, root primitives.MerkleSha256, input *services.CommitStateDiffInput) error {
	persistedBlockHeight := s.height - primitives.BlockHeight(len(s.incCache))
	distance := s.config.StateStorageHistoryRetentionDistance()

	if height > persistedBlockHeight+primitives.BlockHeight(distance) {
		d := s.incCache[0]
		err := s.persistence.Write(d.height, d.ts, d.merkleRoot, d.diff)
		if err != nil {
			return err
		}
		s.incCache = s.incCache[1:]
	}
	s.incCache = append(s.incCache, stateIncrement{
		diff:       _newChainDiff(input.ContractStateDiffs),
		merkleRoot: root,
		height:     height,
		ts:         ts,
	})
	return nil
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
	for i := len(s.incCache) - 1; i >= 0; i-- {
		if s.incCache[i].height > height {
			continue
		}
		if record, exists := s.incCache[i].diff[contract][key]; exists {
			return record, !isZeroValue(record.Value()), nil // cached state increments must include zero values
		}
	}
	return s.persistence.Read(contract, key)
}

func (s *service) _readStateHash(height primitives.BlockHeight) (primitives.MerkleSha256, error) {
	persistedHeight, _, persistedHash, err := s.persistence.ReadMetadata()
	if err != nil {
		return nil, err
	}
	if height == persistedHeight {
		return persistedHash, nil
	}

	cacheIdx := height - persistedHeight - 1
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

	persistedHeight := s.height - primitives.BlockHeight(len(s.incCache))
	if input.BlockHeight < persistedHeight {
		return nil, errors.Errorf("unsupported block height: block %v too old. currently at %v. keeping %v back", input.BlockHeight, s.height, primitives.BlockHeight(s.config.StateStorageHistoryRetentionDistance()))
	}

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
