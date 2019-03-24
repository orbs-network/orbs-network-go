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

type rejectedTransaction struct {
	hash   primitives.Sha256
	status protocol.TransactionStatus
}

type transactionBatch struct {
	maxNumOfTransactions uint32
	sizeLimit            uint32
	logger               log.BasicLogger

	incomingTransactions    Transactions
	transactionsToReject    []*rejectedTransaction
	transactionsForPreOrder Transactions
	validTransactions       Transactions
}

type batchFetcher interface {
	getBatch(maxNumOfTransactions uint32, sizeLimitInBytes uint32) Transactions
}

type batchValidator interface {
	ValidateTransactionForOrdering(transaction *protocol.SignedTransaction, proposedBlockTimestamp primitives.TimestampNano) *ErrTransactionRejected
}

type committedTransactionChecker interface {
	has(txHash primitives.Sha256) bool
}

type preOrderValidator interface {
	preOrderCheck(ctx context.Context, txs Transactions, currentBlockHeight primitives.BlockHeight, currentBlockTimestamp primitives.TimestampNano) ([]protocol.TransactionStatus, error)
}

type vmPreOrderValidator struct {
	vm services.VirtualMachine
}

type txRemover interface {
	remove(ctx context.Context, txHash primitives.Sha256, removalReason protocol.TransactionStatus) *primitives.NodeAddress
}

func (v *vmPreOrderValidator) preOrderCheck(ctx context.Context, txs Transactions, currentBlockHeight primitives.BlockHeight, currentBlockTimestamp primitives.TimestampNano) ([]protocol.TransactionStatus, error) {
	output, err := v.vm.TransactionSetPreOrder(ctx, &services.TransactionSetPreOrderInput{
		SignedTransactions:    txs,
		CurrentBlockHeight:    currentBlockHeight,
		CurrentBlockTimestamp: currentBlockTimestamp,
	})

	if err != nil {
		return nil, err
	}

	return output.PreOrderResults, nil
}

func newTransactionBatch(logger log.BasicLogger, transactions Transactions) *transactionBatch {
	return &transactionBatch{
		logger:               logger,
		incomingTransactions: transactions,
	}
}

func (s *service) GetTransactionsForOrdering(ctx context.Context, input *services.GetTransactionsForOrderingInput) (*services.GetTransactionsForOrderingOutput, error) {

	//TODO(v1) fail if requested block height is in the past
	s.logger.Info("GetTransactionsForOrdering start", trace.LogFieldFrom(ctx), log.BlockHeight(input.CurrentBlockHeight), log.Stringable("transaction-pool-time-between-empty-blocks", s.config.TransactionPoolTimeBetweenEmptyBlocks()))

	// close first  block immediately without waiting (important for gamma)
	if input.CurrentBlockHeight == 1 {
		return &services.GetTransactionsForOrderingOutput{
			SignedTransactions:     nil,
			ProposedBlockTimestamp: proposeBlockTimestampWithCurrentTime(input.PrevBlockTimestamp),
		}, nil
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.BlockTrackerGraceTimeout())
	defer cancel()

	// we're collecting transactions for a new proposed block at input.CurrentBlockHeight
	// wait for previous block height to be synced to avoid processing any tx that was already committed a second time.
	if err := s.blockTracker.WaitForBlock(timeoutCtx, input.CurrentBlockHeight-1); err != nil {
		return nil, err
	}

	pov := &vmPreOrderValidator{vm: s.virtualMachine}

	timeoutCtx, cancel = context.WithTimeout(ctx, s.config.TransactionPoolTimeBetweenEmptyBlocks())
	defer cancel()

	runBatch := func(proposedBlockTimestamp primitives.TimestampNano) (*transactionBatch, error) {
		batch := &transactionBatch{
			logger:               s.logger,
			maxNumOfTransactions: input.MaxNumberOfTransactions,
			sizeLimit:            input.MaxTransactionsSetSizeKb * 1024,
		}
		batch.fetchUsing(s.pendingPool)
		batch.filterInvalidTransactions(ctx, s.validationContext, s.committedPool, proposedBlockTimestamp)
		return batch, batch.runPreOrderValidations(ctx, pov, input.CurrentBlockHeight, proposedBlockTimestamp)
	}

	proposedBlockTimestamp := proposeBlockTimestampWithCurrentTime(input.PrevBlockTimestamp)
	batch, err := runBatch(proposedBlockTimestamp)
	if !batch.hasEnoughTransactions(1) {
		if s.transactionWaiter.waitForIncomingTransaction(timeoutCtx) {
			// propose a new time since we've been waiting
			proposedBlockTimestamp = proposeBlockTimestampWithCurrentTime(input.PrevBlockTimestamp)
			batch, err = runBatch(proposedBlockTimestamp)
		}
	}

	// even on error we want to reject transactions first to their senders before exiting
	batch.notifyRejections(ctx, s.pendingPool)
	out := &services.GetTransactionsForOrderingOutput{
		SignedTransactions:     batch.validTransactions,
		ProposedBlockTimestamp: proposedBlockTimestamp,
	}

	return out, err
}

func proposeBlockTimestampWithCurrentTime(prevBlockTimestamp primitives.TimestampNano) primitives.TimestampNano {
	return digest.CalcNewBlockTimestamp(prevBlockTimestamp, primitives.TimestampNano(time.Now().UnixNano()))
}

func (r *transactionBatch) filterInvalidTransactions(ctx context.Context, validator batchValidator, committedTransactions committedTransactionChecker, proposedBlockTimestamp primitives.TimestampNano) {
	for _, tx := range r.incomingTransactions {
		txHash := digest.CalcTxHash(tx.Transaction())
		if err := validator.ValidateTransactionForOrdering(tx, proposedBlockTimestamp); err != nil {
			r.logger.Info("dropping invalid transaction", log.Error(err), log.String("flow", "checkpoint"), log.Transaction(txHash))
			r.reject(txHash, err.TransactionStatus)
		} else if committedTransactions.has(txHash) {
			r.logger.Info("dropping committed transaction", log.String("flow", "checkpoint"), log.Transaction(txHash))
			r.reject(txHash, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED)
		} else {
			r.queueForPreOrderValidation(tx)
		}
	}

	r.incomingTransactions = nil

	return
}

func (r *transactionBatch) reject(txHash primitives.Sha256, transactionStatus protocol.TransactionStatus) {
	r.transactionsToReject = append(r.transactionsToReject, &rejectedTransaction{txHash, transactionStatus})
}

func (r *transactionBatch) queueForPreOrderValidation(transaction *protocol.SignedTransaction) {
	r.transactionsForPreOrder = append(r.transactionsForPreOrder, transaction)
}

func (r *transactionBatch) notifyRejections(ctx context.Context, remover txRemover) {
	for _, rejected := range r.transactionsToReject {
		remover.remove(ctx, rejected.hash, rejected.status) // TODO(v1) make it a single call and asynchronous - it might speed up the system
	}
	r.transactionsToReject = nil
}

func (r *transactionBatch) accept(transaction *protocol.SignedTransaction) {
	r.validTransactions = append(r.validTransactions, transaction)
}

func (r *transactionBatch) runPreOrderValidations(ctx context.Context, validator preOrderValidator, currentBlockHeight primitives.BlockHeight, currentBlockTimestamp primitives.TimestampNano) error {
	preOrderResults, err := validator.preOrderCheck(ctx, r.transactionsForPreOrder, currentBlockHeight, currentBlockTimestamp)

	if len(preOrderResults) != len(r.transactionsForPreOrder) {
		panic(errors.Errorf("BUG: sent %d transactions for pre-order check and got %d statuses", len(r.transactionsForPreOrder), len(preOrderResults)).Error())
	}

	for i, tx := range r.transactionsForPreOrder {
		if preOrderResults[i] == protocol.TRANSACTION_STATUS_PRE_ORDER_VALID {
			r.accept(tx)
		} else {
			txHash := digest.CalcTxHash(tx.Transaction())
			r.logger.Info("dropping transaction that failed pre-order validation", log.String("flow", "checkpoint"), log.Transaction(txHash))
			r.reject(txHash, preOrderResults[i])
		}
	}

	r.transactionsForPreOrder = nil

	return err
}

func (r *transactionBatch) hasEnoughTransactions(numOfTransactions int) bool {
	return len(r.validTransactions) >= numOfTransactions
}

func (r *transactionBatch) fetchUsing(fetcher batchFetcher) {
	r.incomingTransactions = fetcher.getBatch(r.maxNumOfTransactions, r.sizeLimit)

}
