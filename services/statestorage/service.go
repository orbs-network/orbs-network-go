package statestorage

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
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
	QuerySyncGraceTimeoutMillis() uint64
}

type service struct {
	config Config

	mutex                    *sync.RWMutex
	persistence              adapter.StatePersistence
	lastCommittedBlockHeader *protocol.ResultsBlockHeader

	nextBlockLatch chan int
}

func NewStateStorage(config Config, persistence adapter.StatePersistence) services.StateStorage {
	return &service{
		config:                   config,
		mutex:                    &sync.RWMutex{},
		persistence:              persistence,
		lastCommittedBlockHeader: (&protocol.ResultsBlockHeaderBuilder{}).Build(), // TODO change when system inits genesis block and saves it
		nextBlockLatch:           make(chan int),
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

	s.persistence.WriteState(committedBlock, input.ContractStateDiffs)
	s.lastCommittedBlockHeader = input.ResultsBlockHeader
	height := committedBlock + 1

	prevLatch := s.nextBlockLatch
	s.nextBlockLatch = make(chan int)
	close(prevLatch)

	return &services.CommitStateDiffOutput{NextDesiredBlockHeight: height}, nil
}

func (s *service) ReadKeys(input *services.ReadKeysInput) (*services.ReadKeysOutput, error) {
	if input.ContractName == "" {
		return nil, fmt.Errorf("missing contract name")
	}

	if input.BlockHeight+primitives.BlockHeight(s.config.StateHistoryRetentionInBlockHeights()) <= s.lastCommittedBlockHeader.BlockHeight() {
		return nil, fmt.Errorf("unsupported block height: block %v too old. currently at %v. keeping %v back", input.BlockHeight, s.lastCommittedBlockHeader.BlockHeight(), primitives.BlockHeight(s.config.StateHistoryRetentionInBlockHeights()))
	}

	if !s.checkBlockHeightWithGrace(input.BlockHeight) {
		return nil, fmt.Errorf("unsupported block height: block %v is not yet committed", input.BlockHeight)
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

func (s *service) checkBlockHeightWithGrace(requestedHeight primitives.BlockHeight) bool {
	currentHeight := s.lastCommittedBlockHeader.BlockHeight()
	if currentHeight >= requestedHeight { // requested block already committed
		return true
	}

	graceDist := primitives.BlockHeight(s.config.QuerySyncGraceBlockDist())
	if currentHeight < requestedHeight-graceDist { // requested block too far ahead, no grace
		return false
	}

	timeout := time.NewTimer(time.Duration(s.config.QuerySyncGraceTimeoutMillis()) * time.Millisecond)
	defer timeout.Stop()

	for s.lastCommittedBlockHeader.BlockHeight() < requestedHeight { // sit on latch until desired height or t.o.
		select {
		case <-timeout.C:
			return false
		case <-s.nextBlockLatch:
		}
	}
	return true
}

func (s *service) GetStateStorageBlockHeight(input *services.GetStateStorageBlockHeightInput) (*services.GetStateStorageBlockHeightOutput, error) {
	result := &services.GetStateStorageBlockHeightOutput{
		LastCommittedBlockHeight:    s.lastCommittedBlockHeader.BlockHeight(),
		LastCommittedBlockTimestamp: s.lastCommittedBlockHeader.Timestamp(),
	}
	return result, nil
}

func (s *service) GetStateHash(input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {
	panic("Not implemented")
}
