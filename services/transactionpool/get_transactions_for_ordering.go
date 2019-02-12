package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
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
	validateTransaction(tx *protocol.SignedTransaction) *ErrTransactionRejected
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
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.BlockTrackerGraceTimeout())
	defer cancel()

	// we're collecting transactions for a new proposed block at input.CurrentBlockHeight
	// wait for previous block height to be synced to avoid processing any tx that was already committed a second time.
	if err := s.blockTracker.WaitForBlock(timeoutCtx, input.CurrentBlockHeight-1); err != nil {
		return nil, err
	}

	vctx := s.createValidationContext()
	pov := &vmPreOrderValidator{vm: s.virtualMachine}

	timeoutCtx, cancel = context.WithTimeout(ctx, s.config.TransactionPoolTimeBetweenEmptyBlocks())
	defer cancel()

	runBatch := func() (*transactionBatch, error) {
		batch := &transactionBatch{
			logger:               s.logger,
			maxNumOfTransactions: input.MaxNumberOfTransactions,
			sizeLimit:            input.MaxTransactionsSetSizeKb * 1024,
		}
		batch.fetchUsing(s.pendingPool)
		batch.filterInvalidTransactions(ctx, vctx, s.committedPool)
		return batch, batch.runPreOrderValidations(ctx, pov, input.CurrentBlockHeight, input.CurrentBlockTimestamp)
	}

	minNumOfTransactions := 1
	batch, err := runBatch()
	if !batch.hasEnoughTransactions(minNumOfTransactions) {
		if <-s.transactionWaiter.waitFor(timeoutCtx, minNumOfTransactions) {
			batch, err = runBatch()
		}
	}

	// even on error we want to reject transactions first to their senders before exiting
	batch.notifyRejections(ctx, s.pendingPool)
	out := &services.GetTransactionsForOrderingOutput{SignedTransactions: batch.validTransactions}

	return out, err
}

func (r *transactionBatch) filterInvalidTransactions(ctx context.Context, validator batchValidator, committedTransactions committedTransactionChecker) {
	for _, tx := range r.incomingTransactions {
		txHash := digest.CalcTxHash(tx.Transaction())
		if err := validator.validateTransaction(tx); err != nil {
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
			r.reject(txHash, protocol.TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER)
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
