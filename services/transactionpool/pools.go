package transactionpool

import (
	"sync"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type pendingTransaction struct {
	size	uint32
}

type pendingTxPool struct {
	currentSizeInBytes uint32
	transactions       map[string]*pendingTransaction
	lock               *sync.Mutex

	config Config
}

func (p pendingTxPool) add(transaction *protocol.SignedTransaction) error {
	key := digest.CalcTxHash(transaction.Transaction()).KeyForMap()
	p.lock.Lock()
	defer p.lock.Unlock()
	size := uint32(len(transaction.Raw()))

	if p.currentSizeInBytes + size > p.config.PendingPoolSizeInBytes() {
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_RESERVED} //TODO change to TRANSACTION_STATUS_REJECTED_CONGESTION
	}

	p.currentSizeInBytes += size
	p.transactions[key] = &pendingTransaction{size}
	return nil
}

func (p pendingTxPool) has(transaction *protocol.SignedTransaction) bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	key := digest.CalcTxHash(transaction.Transaction()).KeyForMap()
	_, ok := p.transactions[key]
	return ok
}

func (p pendingTxPool) remove(txhash primitives.Sha256) {
	p.lock.Lock()
	defer p.lock.Unlock()
	_, ok := p.transactions[txhash.KeyForMap()]
	if ok {
		delete(p.transactions, txhash.KeyForMap())
		//p.currentSizeInBytes -= pendingTx.size
	}
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
