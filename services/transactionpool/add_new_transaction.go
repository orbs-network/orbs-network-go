// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"time"
)

func (s *service) AddNewTransaction(ctx context.Context, input *services.AddNewTransactionInput) (*services.AddNewTransactionOutput, error) {
	// too many concurrent AddNewTransaction goroutines from clients may choke the txPool and prevent CommitTransactionReceipts from being handled
	s.addNewTransactionConcurrencyLimiter.RequestSlot()
	defer s.addNewTransactionConcurrencyLimiter.ReleaseSlot()

	txHash := digest.CalcTxHash(input.SignedTransaction.Transaction())

	logger := s.logger.WithTags(log.Transaction(txHash), trace.LogFieldFrom(ctx), log.Stringable("transaction", input.SignedTransaction))

	currentTime := time.Now()
	lastCommittedBlockHeight, lastCommittedBlockTimestamp := s.lastCommittedBlockHeightAndTime()

	if err := s.validationContext.ValidateAddedTransaction(input.SignedTransaction, currentTime, lastCommittedBlockTimestamp); err != nil {
		logger.LogFailedExpectation("transaction is invalid", err.Expected, err.Actual, log.Error(err), log.BlockHeight(lastCommittedBlockHeight), log.TimestampNano("last-committed", lastCommittedBlockTimestamp))
		return s.addTransactionOutputFor(nil, err.TransactionStatus), err
	}

	// TK: this was originally after the check in the committed pool but moved here to make s.addCommitLock more fine grained
	if err := s.validateSingleTransactionForPreOrder(ctx, input.SignedTransaction); err != nil {
		status := protocol.TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER // TODO(https://github.com/orbs-network/orbs-network-go/issues/1017): change to system error
		if errRejected, ok := err.(*ErrTransactionRejected); ok {
			status = errRejected.TransactionStatus
		}
		logger.Error("error validating transaction for preorder", log.Error(err))
		// TODO: add metric here
		return s.addTransactionOutputFor(nil, status), err
	}

	// TK: this was originally in the body of this function but extracted to a function to make s.addCommitLock more fine grained
	output, err := s.addToPendingPoolAfterCheckingCommitted(input.SignedTransaction, txHash, logger)
	if output != nil {
		return output, err
	}

	logger.Info("adding new transaction to the pool", log.String("flow", "checkpoint"))

	s.transactionForwarder.submit(input.SignedTransaction)

	return s.addTransactionOutputFor(nil, protocol.TRANSACTION_STATUS_PENDING), nil
}

func (s *service) addToPendingPoolAfterCheckingCommitted(tx *protocol.SignedTransaction, txHash primitives.Sha256, logger log.BasicLogger) (*services.AddNewTransactionOutput, error) {
	// TODO(https://github.com/orbs-network/orbs-network-go/issues/1020): improve addCommitLock workaround
	s.addCommitLock.RLock()
	defer s.addCommitLock.RUnlock()

	if alreadyCommitted := s.committedPool.get(txHash); alreadyCommitted != nil {
		logger.Info("transaction already committed")
		return s.addTransactionOutputFor(alreadyCommitted.receipt, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED), nil
	}

	address := s.config.NodeAddress()
	if _, err := s.pendingPool.add(tx, address); err != nil {
		logger.Error("error adding transaction to pending pool", log.Error(err))
		return s.addTransactionOutputFor(nil, err.TransactionStatus), err
	}

	return nil, nil
}

func (s *service) validateSingleTransactionForPreOrder(ctx context.Context, transaction *protocol.SignedTransaction) error {
	lastCommittedBlockHeight, _ := s.lastCommittedBlockHeightAndTime()

	// the real pre order checks will run during consensus on some future new block, try to estimate its height and timestamp as closely as possible
	estimatedCurrentBlockHeight := lastCommittedBlockHeight + 1
	estimatedCurrentBlockTimestamp := primitives.TimestampNano(time.Now().UnixNano())

	preOrderCheckResults, err := s.virtualMachine.TransactionSetPreOrder(ctx, &services.TransactionSetPreOrderInput{
		SignedTransactions:    Transactions{transaction},
		CurrentBlockHeight:    estimatedCurrentBlockHeight,
		CurrentBlockTimestamp: estimatedCurrentBlockTimestamp,
	})
	if err != nil {
		return err
	}

	if len(preOrderCheckResults.PreOrderResults) != 1 {
		return errors.Errorf("expected exactly one result from pre-order check, got %+v", preOrderCheckResults)
	}

	if preOrderCheckResults.PreOrderResults[0] != protocol.TRANSACTION_STATUS_PRE_ORDER_VALID {
		return &ErrTransactionRejected{TransactionStatus: preOrderCheckResults.PreOrderResults[0]}
	}

	return nil
}

func (s *service) addTransactionOutputFor(maybeReceipt *protocol.TransactionReceipt, status protocol.TransactionStatus) *services.AddNewTransactionOutput {
	bh, ts := s.lastCommittedBlockHeightAndTime()
	return &services.AddNewTransactionOutput{
		TransactionReceipt: maybeReceipt,
		TransactionStatus:  status,
		BlockHeight:        bh,
		BlockTimestamp:     ts,
	}
}
