package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type rejectedTransaction struct {
	hash   primitives.Sha256
	status protocol.TransactionStatus
}

type ongoingResult struct {
	incomingTransactions    Transactions
	transactionsToReject    []*rejectedTransaction
	transactionsForPreOrder Transactions
	validTransactions       Transactions

	logger log.BasicLogger
}

func (s *service) GetTransactionsForOrdering(ctx context.Context, input *services.GetTransactionsForOrderingInput) (*services.GetTransactionsForOrderingOutput, error) {

	//TODO(v1) fail if requested block height is in the past
	s.logger.Info("GetTransactionsForOrdering called for block height", trace.LogFieldFrom(ctx), log.BlockHeight(input.CurrentBlockHeight))

	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.BlockTrackerGraceTimeout())
	defer cancel()

	// we're collecting transactions for a new proposed block at input.CurrentBlockHeight
	// wait for previous block height to be synced to avoid processing any tx that was already committed a second time.
	if err := s.blockTracker.WaitForBlock(timeoutCtx, input.CurrentBlockHeight-1); err != nil {
		return nil, err
	}

	transactions := s.pendingPool.getBatch(input.MaxNumberOfTransactions, input.MaxTransactionsSetSizeKb*1024)
	vctx := s.createValidationContext()

	ongoing := &ongoingResult{
		logger:               s.logger,
		incomingTransactions: transactions,
	}

	ongoing.filterInvalidTransactions(ctx, vctx, s.committedPool)

	err := ongoing.runPreOrderValidations(ctx, s.virtualMachine, input.CurrentBlockHeight, input.CurrentBlockTimestamp)

	ongoing.notifyRejections(ctx, s.pendingPool)

	out := &services.GetTransactionsForOrderingOutput{SignedTransactions: ongoing.validTransactions}

	return out, err
}

func (r *ongoingResult) filterInvalidTransactions(ctx context.Context, vctx *validationContext, committedTransactions *committedTxPool) {
	for _, tx := range r.incomingTransactions {
		txHash := digest.CalcTxHash(tx.Transaction())
		if err := vctx.validateTransaction(tx); err != nil {
			r.logger.Info("dropping invalid transaction", log.Error(err), log.String("flow", "checkpoint"), log.Transaction(txHash))
			r.reject(txHash, err.TransactionStatus)
		} else if alreadyCommitted := committedTransactions.get(txHash); alreadyCommitted != nil {
			r.logger.Info("dropping committed transaction", log.String("flow", "checkpoint"), log.Transaction(txHash))
			r.reject(txHash, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED)
		} else {
			r.queueForPreOrderValidation(tx)
		}
	}

	r.incomingTransactions = nil

	return
}

func (r *ongoingResult) reject(txHash primitives.Sha256, transactionStatus protocol.TransactionStatus) {
	r.transactionsToReject = append(r.transactionsToReject, &rejectedTransaction{txHash, transactionStatus})
}

func (r *ongoingResult) queueForPreOrderValidation(transaction *protocol.SignedTransaction) {
	r.transactionsForPreOrder = append(r.transactionsForPreOrder, transaction)
}

func (r *ongoingResult) notifyRejections(ctx context.Context, pendingPool *pendingTxPool) {
	for _, rejected := range r.transactionsToReject {
		pendingPool.remove(ctx, rejected.hash, rejected.status) // TODO(v1) make it a single call
	}
}

func (r *ongoingResult) accept(transaction *protocol.SignedTransaction) {
	r.validTransactions = append(r.validTransactions, transaction)
}

func (r *ongoingResult) runPreOrderValidations(ctx context.Context, vm services.VirtualMachine, currentBlockHeight primitives.BlockHeight, currentBlockTimestamp primitives.TimestampNano) error {
	output, err := vm.TransactionSetPreOrder(ctx, &services.TransactionSetPreOrderInput{
		SignedTransactions:    r.transactionsForPreOrder,
		CurrentBlockHeight:    currentBlockHeight,
		CurrentBlockTimestamp: currentBlockTimestamp,
	})

	// even on error we want to reject transactions first to their senders before exiting
	//TODO (v1) what if I got back a different number of transactions than what I sent
	for i, tx := range r.transactionsForPreOrder {
		if output.PreOrderResults[i] == protocol.TRANSACTION_STATUS_PRE_ORDER_VALID {
			r.accept(tx)
		} else {
			txHash := digest.CalcTxHash(tx.Transaction())
			r.logger.Info("dropping transaction that failed pre-order validation", log.String("flow", "checkpoint"), log.Transaction(txHash))
			r.reject(txHash, protocol.TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER)
		}
	}

	return err
}
