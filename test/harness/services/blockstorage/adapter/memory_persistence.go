package adapter

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"sync"
)

type InMemoryBlockPersistence interface {
	adapter.BlockPersistence
	WaitForBlocks(count int)
	FailNextBlocks()
	WaitForTransaction(txhash primitives.Sha256) primitives.BlockHeight
}

type blockHeightChan chan primitives.BlockHeight

type inMemoryBlockPersistence struct {
	blockWritten   chan bool
	blockPairs     []*protocol.BlockPairContainer
	failNextBlocks bool

	lock                  *sync.Mutex
	blockHeightsPerTxHash map[string]blockHeightChan
}

func NewInMemoryBlockPersistence() InMemoryBlockPersistence {
	return &inMemoryBlockPersistence{
		blockWritten:   make(chan bool, 10),
		failNextBlocks: false,

		lock: &sync.Mutex{},
		blockHeightsPerTxHash: make(map[string]blockHeightChan),
	}
}

func (bp *inMemoryBlockPersistence) WaitForBlocks(count int) {
	for i := 0; i < count; i++ {
		<-bp.blockWritten
	}
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
	bp.blockWritten <- true

	bp.advertiseAllTransactions(blockPair.TransactionsBlock)

	return nil
}

func (bp *inMemoryBlockPersistence) ReadAllBlocks() []*protocol.BlockPairContainer {
	return bp.blockPairs
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
