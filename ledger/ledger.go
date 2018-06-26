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
	transactions     []types.Transaction
	blockPersistence blockstorage.BlockPersistence
}

func NewLedger(bp blockstorage.BlockPersistence) Ledger {
	return &ledger{blockPersistence: bp}
}

func (l *ledger) AddTransaction(transaction *types.Transaction) {
	if transaction.Invalid {
		return
	}
	l.transactions = append(l.transactions, *transaction)
	l.blockPersistence.WriteBlock(transaction)
}

func (l *ledger) GetState() int {
	sum := 0
	for _, t := range l.transactions {
		sum += t.Value
	}
	return sum

}
