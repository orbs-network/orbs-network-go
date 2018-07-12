package ledger

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
)

type Ledger interface {
	AddTransaction(transaction *protocol.SignedTransaction)
	GetState() uint64
}

type ledger struct {
	blockPersistence adapter.BlockPersistence
}

func NewLedger(bp adapter.BlockPersistence) Ledger {
	return &ledger{blockPersistence: bp}
}

func (l *ledger) AddTransaction(transaction *protocol.SignedTransaction) {
	if transaction.Transaction().InputArgumentsIterator().NextInputArguments().Uint64Value() > 1000 {
		return
	}
	l.blockPersistence.WriteBlock(transaction)
}

func (l *ledger) GetState() uint64 {
	sum := uint64(0)
	for _, t := range l.blockPersistence.ReadAllBlocks() {
		sum += t.Transaction().InputArgumentsIterator().NextInputArguments().Uint64Value()
	}
	return sum

}
