package transactionpool

import (
	"container/list"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"sync"
	"time"
)

func NewPendingPool(pendingPoolSizeInBytes func() uint32) *pendingTxPool {
	return &pendingTxPool{
		pendingPoolSizeInBytes: pendingPoolSizeInBytes,
		transactionsByHash:     make(map[string]*pendingTransaction),
		transactionList:        list.New(),
		lock:                   &sync.RWMutex{},
	}
}

type pendingTransaction struct {
	gatewayPublicKey primitives.Ed25519PublicKey
	transaction      *protocol.SignedTransaction
	listElement      *list.Element
}

type pendingTxPool struct {
	currentSizeInBytes uint32
	transactionsByHash map[string]*pendingTransaction
	transactionList    *list.List
	lock               *sync.RWMutex

	//FIXME get rid of it
	pendingPoolSizeInBytes func() uint32
}

func (p *pendingTxPool) add(transaction *protocol.SignedTransaction, gatewayPublicKey primitives.Ed25519PublicKey) (primitives.Sha256, *ErrTransactionRejected) {
	size := sizeOf(transaction)

	if p.currentSizeInBytes+size > p.pendingPoolSizeInBytes() {
		return nil, &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_CONGESTION}
	}

	key := digest.CalcTxHash(transaction.Transaction())

	p.lock.Lock()
	defer p.lock.Unlock()
	if _, exists := p.transactionsByHash[key.KeyForMap()]; exists {
		return nil, &ErrTransactionRejected{protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING}
	}

	p.currentSizeInBytes += size
	p.transactionsByHash[key.KeyForMap()] = &pendingTransaction{
		transaction:      transaction,
		gatewayPublicKey: gatewayPublicKey,
		listElement:      p.transactionList.PushFront(transaction),
	}

	return key, nil
}

func (p *pendingTxPool) has(transaction *protocol.SignedTransaction) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	key := digest.CalcTxHash(transaction.Transaction()).KeyForMap()
	_, ok := p.transactionsByHash[key]
	return ok
}

func (p *pendingTxPool) remove(txhash primitives.Sha256) *pendingTransaction {
	p.lock.Lock()
	defer p.lock.Unlock()

	pendingTx, ok := p.transactionsByHash[txhash.KeyForMap()]
	if ok {
		delete(p.transactionsByHash, txhash.KeyForMap())
		p.currentSizeInBytes -= sizeOf(pendingTx.transaction)
		p.transactionList.Remove(pendingTx.listElement)
		return pendingTx
	}

	return nil
}

func (p *pendingTxPool) getBatch(maxNumOfTransactions uint32, sizeLimitInBytes uint32) Transactions {
	txs := make(Transactions, 0, maxNumOfTransactions)
	accumulatedSize := uint32(0)

	p.lock.RLock()
	defer p.lock.RUnlock()

	e := p.transactionList.Back()
	for {
		if e == nil {
			break
		}

		if uint32(len(txs)) >= maxNumOfTransactions {
			break
		}

		tx := e.Value.(*protocol.SignedTransaction)
		//
		accumulatedSize += sizeOf(tx)
		if sizeLimitInBytes > 0 && accumulatedSize > sizeLimitInBytes {
			break
		}

		txs = append(txs, tx)

		e = e.Prev()
	}

	return txs
}

func (p *pendingTxPool) get(txHash primitives.Sha256) *protocol.SignedTransaction {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if ptx, ok := p.transactionsByHash[txHash.KeyForMap()]; ok {
		return ptx.transaction
	}

	return nil
}

func (p *pendingTxPool) clearTransactionsOlderThan(time time.Time) {
	p.lock.RLock()
	e := p.transactionList.Back()
	p.lock.RUnlock()

	for {
		if e == nil {
			break
		}

		tx := e.Value.(*protocol.SignedTransaction)

		e = e.Prev()

		if int64(tx.Transaction().Timestamp()) < time.UnixNano() {
			p.remove(digest.CalcTxHash(tx.Transaction()))
		}
	}
}

func sizeOf(transaction *protocol.SignedTransaction) uint32 {
	return uint32(len(transaction.Raw()))
}
