package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
	"time"
)

func (s *service) AddNewTransaction(input *services.AddNewTransactionInput) (*services.AddNewTransactionOutput, error) {

	//TODO extract to config
	vctx := validationContext{
		expiryWindow:                30 * time.Minute,
		lastCommittedBlockTimestamp: primitives.TimestampNano(time.Now().UnixNano()),
		futureTimestampGrace:        3 * time.Minute,
		virtualChainId:              primitives.VirtualChainId(42),
		transactionInPendingPool:    s.isTransactionInPendingPool,
	}
	err := validateTransaction(input.SignedTransaction, vctx)
	if err != nil {
		s.reporting.Info("transaction is invalid", instrumentation.Error(err), instrumentation.Stringable("transaction", input.SignedTransaction))
		return s.anEmptyReceipt(), err
	}

	if alreadyCommitted := s.committedPool.get(input.SignedTransaction); alreadyCommitted != nil {
		s.reporting.Info("transaction already committed", instrumentation.Stringable("transaction", input.SignedTransaction))
		return &services.AddNewTransactionOutput{
			TransactionReceipt: alreadyCommitted.receipt,
			TransactionStatus:  protocol.TRANSACTION_STATUS_DUPLCIATE_TRANSACTION_ALREADY_COMMITTED,
			//TODO other fields
		}, nil
	}

	if err := s.validateSingleTransactionForPreOrder(input.SignedTransaction); err != nil {
		return s.anEmptyReceipt(), err
	}

	s.reporting.Info("adding new transaction to the pool", instrumentation.Stringable("transaction", input.SignedTransaction))
	if _, err := s.pendingPool.add(input.SignedTransaction); err != nil {
		return nil, err

	}
	s.pendingTransactions <- input.SignedTransaction //TODO remove this

	//TODO batch
	s.forwardTransaction(input.SignedTransaction)

	return &services.AddNewTransactionOutput{}, nil
}

func (s *service) forwardTransaction(tx *protocol.SignedTransaction) error {
	_, err := s.gossip.BroadcastForwardedTransactions(&gossiptopics.ForwardedTransactionsInput{
		Message: &gossipmessages.ForwardedTransactionsMessage{
			SignedTransactions: []*protocol.SignedTransaction{tx},
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: s.config.NodePublicKey(),
			}).Build(),
		},
	})

	return err
}

func (s *service) validateSingleTransactionForPreOrder(transaction *protocol.SignedTransaction) error {
	//TODO handle error from vm call
	preOrderCheckResults, _ := s.virtualMachine.TransactionSetPreOrder(&services.TransactionSetPreOrderInput{
		SignedTransactions: transactions{transaction},
	})

	if len(preOrderCheckResults.PreOrderResults) != 1 {
		return errors.Errorf("expected exactly one result from pre-order check, got %+v", preOrderCheckResults)
	}

	if preOrderCheckResults.PreOrderResults[0] != protocol.TRANSACTION_STATUS_PENDING {
		return &ErrTransactionRejected{TransactionStatus: preOrderCheckResults.PreOrderResults[0]}
	}

	return nil
}

func (s *service) anEmptyReceipt() *services.AddNewTransactionOutput {
	return &services.AddNewTransactionOutput{}
}
