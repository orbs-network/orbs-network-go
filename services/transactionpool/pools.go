package transactionpool

import (
	"container/list"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"sync"
)

func NewPendingPool(config Config) *pendingTxPool {
	return &pendingTxPool{
		config:             config,
		transactionsByHash: make(map[string]*pendingTransaction),
		transactionList:    list.New(),
		lock:               &sync.RWMutex{},
	}
}

func NewCommittedPool() *committedTxPool {
	return &committedTxPool{
		transactions: make(map[string]*committedTransaction),
		lock:         &sync.Mutex{},
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

	config Config
}

func (p *pendingTxPool) add(transaction *protocol.SignedTransaction, gatewayPublicKey primitives.Ed25519PublicKey) (primitives.Sha256, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	size := sizeOf(transaction)

	if p.currentSizeInBytes+size > p.config.PendingPoolSizeInBytes() {
		return nil, &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_CONGESTION}
	}

	key := digest.CalcTxHash(transaction.Transaction())

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

func (p *pendingTxPool) getBatch(maxNumOfTransactions uint32, sizeLimitInBytes uint32) []*protocol.SignedTransaction {
	txs := make([]*protocol.SignedTransaction, 0, maxNumOfTransactions)
	accumulatedSize := uint32(0)
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
		if uint32(len(txs)) >= maxNumOfTransactions || (sizeLimitInBytes > 0 && accumulatedSize > sizeLimitInBytes) {
			break
		}

		txs = append(txs, tx)

		e = e.Prev()
	}

	return txs
}

type committedTxPool struct {
	transactions map[string]*committedTransaction
	lock         *sync.Mutex
}

func (p *committedTxPool) add(receipt *protocol.TransactionReceipt) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.transactions[receipt.Txhash().KeyForMap()] = &committedTransaction{
		receipt: receipt,
	}
}

func (p *committedTxPool) get(transaction *protocol.SignedTransaction) *committedTransaction {
	key := digest.CalcTxHash(transaction.Transaction()).KeyForMap()

	tx := p.transactions[key]

	return tx
}

type committedTransaction struct {
	receipt *protocol.TransactionReceipt
}

func sizeOf(transaction *protocol.SignedTransaction) uint32 {
	return uint32(len(transaction.Raw()))
}
