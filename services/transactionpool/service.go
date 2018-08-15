package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"time"
)

type Config interface {
	PendingPoolSizeInBytes() uint32
	NodePublicKey() primitives.Ed25519PublicKey
}

type service struct {
	pendingTransactions        chan *protocol.SignedTransaction
	gossip                     gossiptopics.TransactionRelay
	virtualMachine             services.VirtualMachine
	transactionResultsHandlers []handlers.TransactionResultsHandler
	reporting                  instrumentation.BasicLogger
	config                     Config

	lastCommittedBlockHeight primitives.BlockHeight
	pendingPool              *pendingTxPool
	committedPool            *committedTxPool
}

func NewTransactionPool(gossip gossiptopics.TransactionRelay, virtualMachine services.VirtualMachine, config Config, reporting instrumentation.BasicLogger) services.TransactionPool {
	s := &service{
		pendingTransactions: make(chan *protocol.SignedTransaction, 10),
		gossip:              gossip,
		virtualMachine:      virtualMachine,
		config:              config,
		reporting:           reporting.For(instrumentation.Service("transaction-pool")),

		pendingPool:   NewPendingPool(config),
		committedPool: NewCommittedPool(),
	}
	gossip.RegisterTransactionRelayHandler(s)
	return s
}

func (s *service) GetTransactionsForOrdering(input *services.GetTransactionsForOrderingInput) (*services.GetTransactionsForOrderingOutput, error) {
	timeout := time.After(100 * time.Millisecond)
	out := &services.GetTransactionsForOrderingOutput{}
	for {
		select {
		case <-timeout:
			return out, nil
		case tx := <-s.pendingTransactions:
			out.SignedTransactions = append(out.SignedTransactions, tx)
			if uint32(len(out.SignedTransactions)) == input.MaxNumberOfTransactions {
				return out, nil
			}
		}
	}
	return out, nil
}

func (s *service) GetCommittedTransactionReceipt(input *services.GetCommittedTransactionReceiptInput) (*services.GetCommittedTransactionReceiptOutput, error) {
	panic("Not implemented")
}

func (s *service) ValidateTransactionsForOrdering(input *services.ValidateTransactionsForOrderingInput) (*services.ValidateTransactionsForOrderingOutput, error) {
	panic("Not implemented")
}

func (s *service) RegisterTransactionResultsHandler(handler handlers.TransactionResultsHandler) {
	s.transactionResultsHandlers = append(s.transactionResultsHandlers, handler)
}

func (s *service) HandleForwardedTransactions(input *gossiptopics.ForwardedTransactionsInput) (*gossiptopics.EmptyOutput, error) {

	//TODO verify message signature
	for _, tx := range input.Message.SignedTransactions {
		if _, err := s.pendingPool.add(tx, input.Message.Sender.SenderPublicKey()); err == nil {
			//TODO this channel needs to go
			s.pendingTransactions <- tx
		}
	}
	return nil, nil
}

func (s *service) isTransactionInPendingPool(transaction *protocol.SignedTransaction) bool {
	return s.pendingPool.has(transaction)
}
