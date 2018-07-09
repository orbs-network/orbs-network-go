package blockstorage

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type Config interface {
	NodeId() string
}

type BlockPersistence interface {
	WriteBlock(transaction *protocol.SignedTransaction)
	ReadAllBlocks() []protocol.SignedTransaction
}

type InMemoryBlockPersistence interface {
	BlockPersistence
	WaitForBlocks(count int)
}

type inMemoryBlockPersistence struct {
	blockWritten chan bool
	transactions []protocol.SignedTransaction
	config       Config
}

func NewInMemoryBlockPersistence(config Config) InMemoryBlockPersistence {
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
