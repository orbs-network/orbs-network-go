package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
)

type InMemoryBlockPersistence interface {
	adapter.BlockPersistence
	WaitForBlocks(count int)
}

type inMemoryBlockPersistence struct {
	blockWritten chan bool
	transactions []protocol.SignedTransaction
	config       adapter.Config
}

func NewInMemoryBlockPersistence(config adapter.Config) InMemoryBlockPersistence {
	return &inMemoryBlockPersistence{
		config:         config,
		blockWritten: make(chan bool, 10),
	}
}

func (bp *inMemoryBlockPersistence) WaitForBlocks(count int) {
	for i := 0; i < count; i++ {
		<-bp.blockWritten
	}
}

func (bp *inMemoryBlockPersistence) WriteBlock(transaction *protocol.SignedTransaction) {
	bp.transactions = append(bp.transactions, *transaction)
	bp.blockWritten <- true
}

func (bp *inMemoryBlockPersistence) ReadAllBlocks() []protocol.SignedTransaction {
	return bp.transactions
}