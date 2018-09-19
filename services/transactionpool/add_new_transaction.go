package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
)

func (s *service) AddNewTransaction(input *services.AddNewTransactionInput) (*services.AddNewTransactionOutput, error) {
	txHash := digest.CalcTxHash(input.SignedTransaction.Transaction())

	s.logger.Info("adding new transaction to the pool", log.String("flow", "checkpoint"), log.Stringable("transaction", input.SignedTransaction), log.Stringable("txHash", txHash))

	if err := s.createValidationContext().validateTransaction(input.SignedTransaction); err != nil {
		s.logger.Info("transaction is invalid", log.Error(err), log.Stringable("transaction", input.SignedTransaction), log.Stringable("txHash", txHash))
		return s.addTransactionOutputFor(nil, err.TransactionStatus), err
	}

	if alreadyCommitted := s.committedPool.get(digest.CalcTxHash(input.SignedTransaction.Transaction())); alreadyCommitted != nil {
		s.logger.Info("transaction already committed", log.Stringable("transaction", input.SignedTransaction), log.Stringable("txHash", txHash))
		return s.addTransactionOutputFor(alreadyCommitted.receipt, protocol.TRANSACTION_STATUS_DUPLCIATE_TRANSACTION_ALREADY_COMMITTED), nil
	}

	if err := s.validateSingleTransactionForPreOrder(input.SignedTransaction); err != nil {
		status := protocol.TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER
		s.logger.Error("error validating transaction for preorder", log.Error(err), log.Stringable("transaction", input.SignedTransaction), log.Stringable("txHash", txHash))
		return s.addTransactionOutputFor(nil, status), err
	}

	if _, err := s.pendingPool.add(input.SignedTransaction, s.config.NodePublicKey()); err != nil {
		s.logger.Error("error adding transaction to pending pool", log.Error(err), log.Stringable("transaction", input.SignedTransaction), log.Stringable("txHash", txHash))
		return s.addTransactionOutputFor(nil, err.TransactionStatus), err

	}
	//TODO batch
	if err := s.forwardTransaction(input.SignedTransaction); err != nil {
		s.logger.Error("error forwarding transaction via gossip", log.Error(err), log.Stringable("transaction", input.SignedTransaction), log.Stringable("txHash", txHash))
	}

	return s.addTransactionOutputFor(nil, protocol.TRANSACTION_STATUS_PENDING), nil
}

func (s *service) forwardTransaction(tx *protocol.SignedTransaction) error {
	s.logger.Info("forwarding transaction to peers", log.Stringable("transaction", tx))

	sig, err := signature.SignEd25519(s.config.NodePrivateKey(), tx.Raw())
	if err != nil {
		return err
	}

	_, err = s.gossip.BroadcastForwardedTransactions(&gossiptopics.ForwardedTransactionsInput{
		Message: &gossipmessages.ForwardedTransactionsMessage{
			SignedTransactions: Transactions{tx},
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: s.config.NodePublicKey(),
				Signature:       sig,
			}).Build(),
		},
	})

	return err
}

func (s *service) validateSingleTransactionForPreOrder(transaction *protocol.SignedTransaction) error {
	//TODO handle error from vm call
	preOrderCheckResults, _ := s.virtualMachine.TransactionSetPreOrder(&services.TransactionSetPreOrderInput{
		SignedTransactions: Transactions{transaction},
		BlockHeight:        s.lastCommittedBlockHeight,
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
		TransactionStatus:  status,
		BlockHeight:        s.lastCommittedBlockHeight,
		BlockTimestamp:     s.lastCommittedBlockTimestamp,
	}
}
