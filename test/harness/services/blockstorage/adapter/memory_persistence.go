package adapter

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"sync"
	"time"
)

type InMemoryBlockPersistence interface {
	adapter.BlockPersistence
	FailNextBlocks()
	WaitForTransaction(txhash primitives.Sha256) primitives.BlockHeight
}

type blockHeightChan chan primitives.BlockHeight

type inMemoryBlockPersistence struct {
	blockPairs     []*protocol.BlockPairContainer
	failNextBlocks bool
	tracker        *synchronization.BlockTracker

	lock                  *sync.Mutex
	blockHeightsPerTxHash map[string]blockHeightChan
}

func NewInMemoryBlockPersistence() InMemoryBlockPersistence {
	return &inMemoryBlockPersistence{
		failNextBlocks: false,
		tracker:        synchronization.NewBlockTracker(0, 5, time.Millisecond*100),

		lock: &sync.Mutex{},
		blockHeightsPerTxHash: make(map[string]blockHeightChan),
	}
}

func (bp *inMemoryBlockPersistence) GetBlockTracker() *synchronization.BlockTracker {
	return bp.tracker
}

func (bp *inMemoryBlockPersistence) WaitForTransaction(txhash primitives.Sha256) primitives.BlockHeight {
	h := <-bp.getChanFor(txhash)
	return h
}

func (bp *inMemoryBlockPersistence) WriteBlock(blockPair *protocol.BlockPairContainer) error {
	if bp.failNextBlocks {
		return errors.New("could not write a block")
	}

	bp.blockPairs = append(bp.blockPairs, blockPair)
	bp.tracker.IncrementHeight()

	bp.advertiseAllTransactions(blockPair.TransactionsBlock)

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

func (bp *inMemoryBlockPersistence) getChanFor(txhash primitives.Sha256) blockHeightChan {
	bp.lock.Lock()
	defer bp.lock.Unlock()

	ch, ok := bp.blockHeightsPerTxHash[txhash.KeyForMap()]
	if !ok {
		ch = make(blockHeightChan, 1)
		bp.blockHeightsPerTxHash[txhash.KeyForMap()] = ch
	}

	return ch
}
func (bp *inMemoryBlockPersistence) advertiseAllTransactions(block *protocol.TransactionsBlockContainer) {
	for _, tx := range block.SignedTransactions {
		bp.getChanFor(digest.CalcTxHash(tx.Transaction())) <- block.Header.BlockHeight()
	}
}

func (bp *inMemoryBlockPersistence) GetLastBlock() (*protocol.BlockPairContainer, error) {
	count := len(bp.blockPairs)

	if count == 0 {
		return nil, nil
	}

	return bp.blockPairs[count-1], nil
}
