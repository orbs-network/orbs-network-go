package transactionpool

import (
	"sync"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type pendingTxPool struct {
	transactions map[string]bool
	lock         *sync.Mutex
	config       Config
}

func (p pendingTxPool) add(transaction *protocol.SignedTransaction) {
	key := digest.CalcTxHash(transaction.Transaction()).KeyForMap()
	p.lock.Lock()
	defer p.lock.Unlock()
	p.transactions[key] = true
}

func (p pendingTxPool) has(transaction *protocol.SignedTransaction) bool {
	key := digest.CalcTxHash(transaction.Transaction()).KeyForMap()
	ok, _ := p.transactions[key]
	return ok
}

func (p pendingTxPool) remove(txhash primitives.Sha256) {
	delete(p.transactions, txhash.KeyForMap())
}

type committedTxPool struct {
	transactions map[string]*committedTransaction
	lock         *sync.Mutex
}

func (p committedTxPool) add(receipt *protocol.TransactionReceipt) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.transactions[receipt.Txhash().KeyForMap()] = &committedTransaction{
		receipt: receipt,
	}
}

func (p committedTxPool) get(transaction *protocol.SignedTransaction) *committedTransaction {
	key := digest.CalcTxHash(transaction.Transaction()).KeyForMap()

	tx := p.transactions[key]

	return tx
}

type committedTransaction struct {
	receipt *protocol.TransactionReceipt
}

