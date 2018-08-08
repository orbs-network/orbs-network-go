package transactionpool

import (
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

func (s *service) AddNewTransaction(input *services.AddNewTransactionInput) (*services.AddNewTransactionOutput, error) {

	//TODO extract to config
	vctx := validationContext{
		expiryWindow:                30 * time.Minute,
		lastCommittedBlockTimestamp: primitives.TimestampNano(time.Now().UnixNano()),
		futureTimestampGrace:        3 * time.Minute,
		virtualChainId:              primitives.VirtualChainId(42),
		transactionInPendingPool:    s.isTransactionInPendingPool,
		transactionInCommittedPool:  s.isTransactionInCommittedPool,
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
	s.pendingTransactions <- input.SignedTransaction //TODO remove this
	s.pendingPool.add(input.SignedTransaction)

	return &services.AddNewTransactionOutput{}, nil
}
