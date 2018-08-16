package adapter

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"time"
)

type InMemoryBlockPersistence interface {
	adapter.BlockPersistence
	WaitForBlocks(count int)
	FailNextBlocks()
}

type inMemoryBlockPersistence struct {
	blockWritten   chan bool
	blockPairs     []*protocol.BlockPairContainer
	failNextBlocks bool
}

func NewInMemoryBlockPersistence() InMemoryBlockPersistence {
	return &inMemoryBlockPersistence{
		blockWritten:   make(chan bool, 10),
		failNextBlocks: false,
	}
}

func (bp *inMemoryBlockPersistence) WaitForBlocks(count int) {
	for i := 0; i < count; i++ {
		<-bp.blockWritten
	}
}

func (bp *inMemoryBlockPersistence) WriteBlock(blockPair *protocol.BlockPairContainer) error {
	if bp.failNextBlocks {
		return errors.New("could not write a block")
	}

	bp.blockPairs = append(bp.blockPairs, blockPair)
	bp.blockWritten <- true

	return nil
}

func (bp *inMemoryBlockPersistence) ReadAllBlocks() []*protocol.BlockPairContainer {
	return bp.blockPairs
}

func (bp *inMemoryBlockPersistence) GetReceiptRelevantBlocks(txTimeStamp primitives.TimestampNano, rules adapter.BlockSearchRules) []*protocol.BlockPairContainer {
	start := txTimeStamp - primitives.TimestampNano(rules.StartGraceNano)
	end := txTimeStamp + primitives.TimestampNano(rules.EndGraceNano+rules.TransactionExpireNano)

	if end < start {
		return nil
	}
	var relevantBlocks []*protocol.BlockPairContainer
	interval := end - start
	// TODO: FIXME: sanity check, this is really useless here right now, but we are going to refactor this in about two-three weeks, and when we do, this is here to remind us to have a sanity check on this query
	if interval > primitives.TimestampNano(time.Hour.Nanoseconds()) {
		return nil
	}

	for _, b := range bp.blockPairs {
		delta := end - b.TransactionsBlock.Header.Timestamp()
		if delta > 0 && interval > delta {
			relevantBlocks = append(relevantBlocks, b)
		}
	}
	return relevantBlocks
}

func (bp *inMemoryBlockPersistence) GetTransactionsBlock(height primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error) {
	for _, bp := range bp.blockPairs {
		if bp.TransactionsBlock.Header.BlockHeight() == height {
			return bp.TransactionsBlock, nil
		}
	}

	return nil, fmt.Errorf("transactions block header with height %v not found", height)
}

func (bp *inMemoryBlockPersistence) GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error) {
	for _, bp := range bp.blockPairs {
		if bp.TransactionsBlock.Header.BlockHeight() == height {
			return bp.ResultsBlock, nil
		}
	}

	return nil, fmt.Errorf("results block header with height %v not found", height)
}

func (bp *inMemoryBlockPersistence) FailNextBlocks() {
	bp.failNextBlocks = true
}
