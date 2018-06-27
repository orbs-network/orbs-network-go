package blockstorage

import (
	"github.com/orbs-network/orbs-network-go/types"
)

type BlockPersistence interface {
	WriteBlock(transaction *types.Transaction)
	ReadAllBlocks() []types.Transaction
}

type InMemoryBlockPersistence interface {
	BlockPersistence
	WaitForBlocks(count int)
}

type inMemoryBlockPersistence struct {
	blockWritten chan bool
	name         string
	transactions []types.Transaction
}

func NewInMemoryBlockPersistence(name string) InMemoryBlockPersistence {
	return &inMemoryBlockPersistence{
		name:         name,
		blockWritten: make(chan bool, 10),
	}
}

func (bp *inMemoryBlockPersistence) WaitForBlocks(count int) {
	for i := 0; i < count; i++ {
		<-bp.blockWritten
	}
}

func (bp *inMemoryBlockPersistence) WriteBlock(transaction *types.Transaction) {
	bp.transactions = append(bp.transactions, *transaction)
	bp.blockWritten <- true
}

func (bp *inMemoryBlockPersistence) ReadAllBlocks() []types.Transaction {
	return bp.transactions
}
