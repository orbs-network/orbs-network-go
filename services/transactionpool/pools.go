package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"sort"
	"sync"
)

func NewPendingPool(config Config) *pendingTxPool {
	return &pendingTxPool{
		config:       config,
		transactions: make(map[string]*pendingTransaction),
		lock:         &sync.RWMutex{},
	}
}

func NewCommittedPool() *committedTxPool {
	return &committedTxPool{
		transactions: make(map[string]*committedTransaction),
		lock:         &sync.Mutex{},
	}
}

type pendingTransaction struct {
	sizeInBytes      uint32
	gatewayPublicKey primitives.Ed25519PublicKey
	transaction      *protocol.SignedTransaction
}

type pendingTxPool struct {
	currentSizeInBytes uint32
	transactions       map[string]*pendingTransaction
	lock               *sync.RWMutex

	config Config
}

func (p *pendingTxPool) add(transaction *protocol.SignedTransaction, gatewayPublicKey primitives.Ed25519PublicKey) (primitives.Sha256, error) {
	key := digest.CalcTxHash(transaction.Transaction())
	p.lock.Lock()
	defer p.lock.Unlock()
	size := uint32(len(transaction.Raw()))

	if p.currentSizeInBytes+size > p.config.PendingPoolSizeInBytes() {
		return nil, &ErrTransactionRejected{protocol.TRANSACTION_STATUS_RESERVED} //TODO change to TRANSACTION_STATUS_REJECTED_CONGESTION
	}

	p.currentSizeInBytes += size
	p.transactions[key.KeyForMap()] = &pendingTransaction{
		transaction:      transaction,
		sizeInBytes:      size,
		gatewayPublicKey: gatewayPublicKey,
	}
	return key, nil
}

func (p *pendingTxPool) has(transaction *protocol.SignedTransaction) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	key := digest.CalcTxHash(transaction.Transaction()).KeyForMap()
	_, ok := p.transactions[key]
	return ok
}

func (p *pendingTxPool) remove(txhash primitives.Sha256) *pendingTransaction {
	p.lock.Lock()
	defer p.lock.Unlock()
	pendingTx, ok := p.transactions[txhash.KeyForMap()]
	if ok {
		delete(p.transactions, txhash.KeyForMap())
		p.currentSizeInBytes -= pendingTx.sizeInBytes
		return pendingTx
	}

	return nil
}

func (p *pendingTxPool) getBatch(maxNumOfTransactions uint32, sizeLimitInBytes uint32) []*protocol.SignedTransaction {
	txs := make([]*protocol.SignedTransaction, 0, maxNumOfTransactions)
	accumulatedSize := uint32(0)
	for _, tx := range p.transactions {
		accumulatedSize += tx.sizeInBytes
		if uint32(len(txs)) >= maxNumOfTransactions || (sizeLimitInBytes > 0 && accumulatedSize > sizeLimitInBytes) {
			break
		}

		txs = append(txs, tx.transaction)
	}

	sort.Slice(txs, func(i, j int) bool {
		return txs[i].Transaction().Timestamp() < txs[j].Transaction().Timestamp()
	})

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
