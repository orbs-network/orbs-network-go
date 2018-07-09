package ledger

import (
	"github.com/orbs-network/orbs-network-go/blockstorage"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type Ledger interface {
	AddTransaction(transaction *protocol.SignedTransaction)
	GetState() uint64
}

type ledger struct {
	blockPersistence blockstorage.BlockPersistence
}

func NewLedger(bp blockstorage.BlockPersistence) Ledger {
	return &ledger{blockPersistence: bp}
}

func (l *ledger) AddTransaction(transaction *protocol.SignedTransaction) {
	if transaction.TransactionContent().InputArgumentIterator().NextInputArgument().TypeUint64() > 1000 {
		return
	}
	l.blockPersistence.WriteBlock(transaction)
}

func (l *ledger) GetState() uint64 {
	sum := uint64(0)
	for _, t := range l.blockPersistence.ReadAllBlocks() {
		sum += t.TransactionContent().InputArgumentIterator().NextInputArgument().TypeUint64()
	}
	return sum

}
