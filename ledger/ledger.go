package ledger

import "github.com/orbs-network/orbs-network-go/types"

type Ledger interface {
	AddTransaction(transaction *types.Transaction)
	GetState() int
}

type ledger struct {
	transactions []types.Transaction
}

func NewLedger() Ledger {
	return &ledger{}
}

func (l *ledger) AddTransaction(transaction *types.Transaction) {
	l.transactions = append(l.transactions, *transaction)
}

func (l *ledger) GetState() int {
	sum := 0
	for _, t := range l.transactions {
		sum += t.Value
	}
	return sum

}
