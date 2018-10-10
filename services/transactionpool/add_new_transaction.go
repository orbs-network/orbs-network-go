package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) AddNewTransaction(input *services.AddNewTransactionInput) (*services.AddNewTransactionOutput, error) {
	txHash := digest.CalcTxHash(input.SignedTransaction.Transaction())

	s.logger.Info("adding new transaction to the pool", log.String("flow", "checkpoint"), log.Stringable("transaction", input.SignedTransaction), log.Stringable("txHash", txHash))

	if err := s.createValidationContext().validateTransaction(input.SignedTransaction); err != nil {
		s.logger.LogFailedExpectation("transaction is invalid", err.Expected, err.Actual, log.Error(err), log.Stringable("transaction", input.SignedTransaction), log.Stringable("txHash", txHash))
		return s.addTransactionOutputFor(nil, err.TransactionStatus), err
	}

	if alreadyCommitted := s.committedPool.get(digest.CalcTxHash(input.SignedTransaction.Transaction())); alreadyCommitted != nil {
		s.logger.Info("transaction already committed", log.Stringable("transaction", input.SignedTransaction), log.Stringable("txHash", txHash))
		return s.addTransactionOutputFor(alreadyCommitted.receipt, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED), nil
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

	s.forwardQueueMutex.Lock()
	defer s.forwardQueueMutex.Unlock()
	s.forwardQueue = append(s.forwardQueue, input.SignedTransaction)

	return s.addTransactionOutputFor(nil, protocol.TRANSACTION_STATUS_PENDING), nil
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
