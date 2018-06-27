package ledger

import (
	"github.com/orbs-network/orbs-network-go/types"
	"github.com/orbs-network/orbs-network-go/blockstorage"
)

type Ledger interface {
	AddTransaction(transaction *types.Transaction)
	GetState() int
}

type ledger struct {
	blockPersistence blockstorage.BlockPersistence
}

func NewLedger(bp blockstorage.BlockPersistence) Ledger {
	return &ledger{blockPersistence: bp}
}

func (l *ledger) AddTransaction(transaction *types.Transaction) {
	if transaction.Invalid {
		return
	}
	l.blockPersistence.WriteBlock(transaction)
}

func (l *ledger) GetState() int {
	sum := 0
	for _, t := range l.blockPersistence.ReadAllBlocks() {
		sum += t.Value
	}
	return sum

}
