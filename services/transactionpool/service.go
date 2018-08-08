package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"time"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type service struct {
	pendingTransactions chan *protocol.SignedTransaction
	gossip              gossiptopics.TransactionRelay
	reporting           instrumentation.BasicLogger
}

func NewTransactionPool(gossip gossiptopics.TransactionRelay, reporting instrumentation.BasicLogger) services.TransactionPool {
	s := &service{
		pendingTransactions: make(chan *protocol.SignedTransaction, 10),
		gossip:              gossip,
		reporting:           reporting.For(instrumentation.Service("transaction-pool")),
	}
	gossip.RegisterTransactionRelayHandler(s)
	return s
}

func (s *service) AddNewTransaction(input *services.AddNewTransactionInput) (*services.AddNewTransactionOutput, error) {

	//TODO extract to config
	vctx := validationContext{
		expiryWindow:                30 * time.Minute,
		lastCommittedBlockTimestamp: primitives.TimestampNano(time.Now().UnixNano()),
		futureTimestampGrace:        3 * time.Minute,
		virtualChainId:              primitives.VirtualChainId(42),
	}
	err := validateTransaction(input.SignedTransaction, vctx)
	if err != nil {
		s.reporting.Info("transaction is invalid", instrumentation.Error(err), instrumentation.Stringable("transaction", input.SignedTransaction))
		return nil, err
	}

	s.gossip.BroadcastForwardedTransactions(&gossiptopics.ForwardedTransactionsInput{
		Message: &gossipmessages.ForwardedTransactionsMessage{

			SignedTransactions: []*protocol.SignedTransaction{input.SignedTransaction},
		},
	})

	s.reporting.Info("adding new transaction to the pool", instrumentation.Stringable("transaction", input.SignedTransaction))
	s.pendingTransactions <- input.SignedTransaction

	return &services.AddNewTransactionOutput{}, nil
}

func (s *service) GetTransactionsForOrdering(input *services.GetTransactionsForOrderingInput) (*services.GetTransactionsForOrderingOutput, error) {
	out := &services.GetTransactionsForOrderingOutput{}
	out.SignedTransactions = make([]*protocol.SignedTransaction, input.MaxNumberOfTransactions)
	for i := uint32(0); i < input.MaxNumberOfTransactions; i++ {
		out.SignedTransactions[i] = <-s.pendingTransactions
	}
	return out, nil
}

func (s *service) GetCommittedTransactionReceipt(input *services.GetCommittedTransactionReceiptInput) (*services.GetCommittedTransactionReceiptOutput, error) {
	panic("Not implemented")
}

func (s *service) ValidateTransactionsForOrdering(input *services.ValidateTransactionsForOrderingInput) (*services.ValidateTransactionsForOrderingOutput, error) {
	panic("Not implemented")
}

func (s *service) CommitTransactionReceipts(input *services.CommitTransactionReceiptsInput) (*services.CommitTransactionReceiptsOutput, error) {
	panic("Not implemented")
}

func (s *service) RegisterTransactionResultsHandler(handler handlers.TransactionResultsHandler) {
	panic("Not implemented")
}

func (s *service) HandleForwardedTransactions(input *gossiptopics.ForwardedTransactionsInput) (*gossiptopics.EmptyOutput, error) {
	for _, tx := range input.Message.SignedTransactions {
		s.pendingTransactions <- tx
	}
	return nil, nil
}
