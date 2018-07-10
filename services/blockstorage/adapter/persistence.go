package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type Config interface {
	NodeId() string
}

type BlockPersistence interface {
	WriteBlock(signedTransaction *protocol.SignedTransaction)
	ReadAllBlocks() []protocol.SignedTransaction
}

type blockPersistence struct {
	blockWritten chan bool
	transactions []protocol.SignedTransaction
	config       Config
}

func NewBlockPersistence(config Config) BlockPersistence {
	return &blockPersistence{
		config:         config,
		blockWritten: make(chan bool, 10),
	}
}

func (bp *blockPersistence) WriteBlock(signedTransaction *protocol.SignedTransaction) {
	bp.transactions = append(bp.transactions, *signedTransaction)
	bp.blockWritten <- true
}

func (bp *blockPersistence) ReadAllBlocks() []protocol.SignedTransaction {
	return bp.transactions
}