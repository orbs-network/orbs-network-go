package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) AddNewTransaction(ctx context.Context, input *services.AddNewTransactionInput) (*services.AddNewTransactionOutput, error) {
	txHash := digest.CalcTxHash(input.SignedTransaction.Transaction())

	logger := s.logger.WithTags(log.Transaction(txHash), trace.LogFieldFrom(ctx), log.Stringable("transaction", input.SignedTransaction))

	logger.Info("adding new transaction to the pool", log.String("flow", "checkpoint"))

	if err := s.createValidationContext().validateTransaction(input.SignedTransaction); err != nil {
		s.logger.LogFailedExpectation("transaction is invalid", err.Expected, err.Actual, log.Error(err))
		return s.addTransactionOutputFor(nil, err.TransactionStatus), err
	}

	if alreadyCommitted := s.committedPool.get(digest.CalcTxHash(input.SignedTransaction.Transaction())); alreadyCommitted != nil {
		logger.Info("transaction already committed")
		return s.addTransactionOutputFor(alreadyCommitted.receipt, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED), nil
	}

	if err := s.validateSingleTransactionForPreOrder(ctx, input.SignedTransaction); err != nil {
		status := protocol.TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER
		logger.Error("error validating transaction for preorder", log.Error(err))
		return s.addTransactionOutputFor(nil, status), err
	}

	if _, err := s.pendingPool.add(input.SignedTransaction, s.config.NodeAddress()); err != nil {
		s.logger.Error("error adding transaction to pending pool", log.Error(err))
		return s.addTransactionOutputFor(nil, err.TransactionStatus), err

	}

	s.transactionForwarder.submit(input.SignedTransaction)

	return s.addTransactionOutputFor(nil, protocol.TRANSACTION_STATUS_PENDING), nil
}

func (s *service) validateSingleTransactionForPreOrder(ctx context.Context, transaction *protocol.SignedTransaction) error {
	bh, _ := s.currentBlockHeightAndTime()
	//TODO(v1) handle error from vm call
	preOrderCheckResults, _ := s.virtualMachine.TransactionSetPreOrder(ctx, &services.TransactionSetPreOrderInput{
		SignedTransactions: Transactions{transaction},
		BlockHeight:        bh,
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
	bh, ts := s.currentBlockHeightAndTime()
	return &services.AddNewTransactionOutput{
		TransactionReceipt: maybeReceipt,
		TransactionStatus:  status,
		BlockHeight:        bh,
		BlockTimestamp:     ts,
	}
}
