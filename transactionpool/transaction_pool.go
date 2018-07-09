package transactionpool

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	gossip2 "github.com/orbs-network/orbs-spec/types/go/services/gossip"
)

type inMemoryTransactionPool struct {
	pendingTransactions chan *protocol.SignedTransaction
}

func NewTransactionPool(relay gossip2.TransactionRelay) services.TransactionPool {
	pool := &inMemoryTransactionPool{make(chan *protocol.SignedTransaction, 10)}
	relay.RegisterTransactionRelayHandler(pool)
	return pool
}

func (p *inMemoryTransactionPool) AddNewTransaction(input *services.AddNewTransactionInput) (*services.AddNewTransactionOutput, error) {
	p.pendingTransactions <- input.SignedTransaction
	return &services.AddNewTransactionOutput{}, nil
}

func (p *inMemoryTransactionPool) GetTransactionsForOrdering(input *services.GetTransactionsForOrderingInput) (*services.GetTransactionsForOrderingOutput, error) {
	out := &services.GetTransactionsForOrderingOutput{}

	out.SignedTransaction = make([]*protocol.SignedTransaction, input.MaxNumberOfTransactions)
	for i := uint32(0); i < input.MaxNumberOfTransactions; i++ {
		out.SignedTransaction[i] = <- p.pendingTransactions
	}

	return out, nil
}

// Deprecated: TransactionListener is going away in favor of TransactionRelayGossipHandler
func (p *inMemoryTransactionPool) OnForwardTransaction(tx *protocol.SignedTransaction) {
	p.pendingTransactions <- tx
}

func (p *inMemoryTransactionPool) HandleForwardedTransactions(input *gossip2.ForwardedTransactionsInput) (*gossip2.TransactionRelayOutput, error) {
	txs := input.Transactions
	for _, tx := range txs {
		p.pendingTransactions <- tx
	}

	return nil, nil
}


func (p *inMemoryTransactionPool) GetCommittedTransactionReceipt(input *services.GetCommittedTransactionReceiptInput) (*services.GetCommittedTransactionReceiptOutput, error) {
	panic("Not implemented")
}

func (p *inMemoryTransactionPool) ValidateTransactionsForOrdering(input *services.ValidateTransactionsForOrderingInput) (*services.ValidateTransactionsForOrderingOutput, error) {
	panic("Not implemented")
}

func (p *inMemoryTransactionPool) CommitTransactionReceipts(input *services.CommitTransactionReceiptsInput) (*services.CommitTransactionReceiptsOutput, error) {
	panic("Not implemented")
}

func (p *inMemoryTransactionPool) RegisterTransactionResultsHandler(handler handlers.TransactionResultsHandler) {
	panic("Not implemented")
}
