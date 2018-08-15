package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
)

func (s *service) AddNewTransaction(input *services.AddNewTransactionInput) (*services.AddNewTransactionOutput, error) {

	err := s.createValidationContext().validateTransaction(input.SignedTransaction)
	if err != nil {
		s.log.Info("transaction is invalid", instrumentation.Error(err), instrumentation.Stringable("transaction", input.SignedTransaction))
		return s.addTransactionOutputFor(nil, err.(*ErrTransactionRejected).TransactionStatus), err
	}

	if s.pendingPool.has(input.SignedTransaction) {
		return nil, &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_DUPLCIATE_PENDING_TRANSACTION}
	}

	if alreadyCommitted := s.committedPool.get(input.SignedTransaction); alreadyCommitted != nil {
		s.log.Info("transaction already committed", instrumentation.Stringable("transaction", input.SignedTransaction))
		return s.addTransactionOutputFor(alreadyCommitted.receipt, protocol.TRANSACTION_STATUS_DUPLCIATE_TRANSACTION_ALREADY_COMMITTED), nil
	}

	if err := s.validateSingleTransactionForPreOrder(input.SignedTransaction); err != nil {
		return s.addTransactionOutputFor(nil, protocol.TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER), err
	}

	s.log.Info("adding new transaction to the pool", instrumentation.Stringable("transaction", input.SignedTransaction))
	if _, err := s.pendingPool.add(input.SignedTransaction, s.config.NodePublicKey()); err != nil {
		s.log.Error("error adding transaction to pending pool", instrumentation.Error(err), instrumentation.Stringable("transaction", input.SignedTransaction))
		return nil, err

	}
	//TODO batch
	s.forwardTransaction(input.SignedTransaction)

	return s.addTransactionOutputFor(nil, protocol.TRANSACTION_STATUS_PENDING), nil
}

func (s *service) forwardTransaction(tx *protocol.SignedTransaction) error {
	// TODO sign
	//sig, err := signature.SignEd25519(s.config.NodePrivateKey(), signedData)
	//if err != nil {
	//	return nil, err
	//}

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
		//TODO send block height
	})

	if len(preOrderCheckResults.PreOrderResults) != 1 {
		return errors.Errorf("expected exactly one result from pre-order check, got %+v", preOrderCheckResults)
	}

	if preOrderCheckResults.PreOrderResults[0] != protocol.TRANSACTION_STATUS_PRE_ORDER_VALID {
		return &ErrTransactionRejected{TransactionStatus: preOrderCheckResults.PreOrderResults[0]}
	}

	return nil
}

func (s *service) addTransactionOutputFor(maybeReceipt *protocol.TransactionReceipt, status protocol.TransactionStatus) *services.AddNewTransactionOutput {
	return &services.AddNewTransactionOutput{
		TransactionReceipt: maybeReceipt,
		TransactionStatus: status,
		BlockHeight: s.lastCommittedBlockHeight,
		BlockTimestamp: s.lastCommittedBlockTimestamp,
	}
}
