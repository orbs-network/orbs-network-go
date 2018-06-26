package blockstorage

import "github.com/orbs-network/orbs-network-go/types"

type BlockPersistence interface {
	WriteBlock(transaction *types.Transaction)
}

type InMemoryBlockPersistence interface {
	BlockPersistence
	WaitForBlocks(count int)
}

type inMemoryBlockPersistence struct {
	blockWritten chan bool
}

func NewInMemoryBlockPersistence() InMemoryBlockPersistence {
	return &inMemoryBlockPersistence{make(chan bool, 10)}
}

func (bp *inMemoryBlockPersistence) WaitForBlocks(count int) {
	for i := 0; i < count; i++ {
		<-bp.blockWritten
	}
}

func (bp *inMemoryBlockPersistence) WriteBlock(transaction *types.Transaction) {
	bp.blockWritten <- true
}
