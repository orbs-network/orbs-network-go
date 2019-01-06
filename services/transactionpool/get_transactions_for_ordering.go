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

	out := &services.GetTransactionsForOrderingOutput{}
	transactions := s.pendingPool.getBatch(input.MaxNumberOfTransactions, input.MaxTransactionsSetSizeKb*1024)
	vctx := s.createValidationContext()

	transactionsForPreOrder, rejectedTransactions := s.filterInvalidTransactions(ctx, input, transactions, vctx)

	output, err := s.virtualMachine.TransactionSetPreOrder(ctx, &services.TransactionSetPreOrderInput{
		SignedTransactions:    transactionsForPreOrder,
		CurrentBlockHeight:    input.CurrentBlockHeight,
		CurrentBlockTimestamp: input.CurrentBlockTimestamp,
	})

	// even on error we want to reject transactions first to their senders before exiting
	if len(output.PreOrderResults) == len(transactionsForPreOrder) { //TODO ask Tal why there's no Else to this If
		for i, tx := range transactionsForPreOrder {
			if output.PreOrderResults[i] == protocol.TRANSACTION_STATUS_PRE_ORDER_VALID {
				out.SignedTransactions = append(out.SignedTransactions, tx)
			} else {
				txHash := digest.CalcTxHash(tx.Transaction())
				s.logger.Info("dropping transaction that failed pre-order validation", log.String("flow", "checkpoint"), log.Transaction(txHash))
				rejectedTransactions = append(rejectedTransactions, &rejectedTransaction{txHash, protocol.TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER})
			}
		}
	}

	s.notifyRejections(ctx, rejectedTransactions)

	return out, err
}

func (s *service) filterInvalidTransactions(ctx context.Context, input *services.GetTransactionsForOrderingInput, transactions Transactions, vctx *validationContext) (valid Transactions, invalid []*rejectedTransaction) {
	for _, tx := range transactions {
		txHash := digest.CalcTxHash(tx.Transaction())
		if err := vctx.validateTransaction(tx); err != nil {
			s.logger.Info("dropping invalid transaction", log.Error(err), log.String("flow", "checkpoint"), log.Transaction(txHash))
			invalid = append(invalid, &rejectedTransaction{txHash, err.TransactionStatus})
		} else if alreadyCommitted := s.committedPool.get(txHash); alreadyCommitted != nil {
			s.logger.Info("dropping committed transaction", log.String("flow", "checkpoint"), log.Transaction(txHash))
			invalid = append(invalid, &rejectedTransaction{txHash, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED})
		} else {
			valid = append(valid, tx)
		}
	}
	return
}

func (s *service) notifyRejections(ctx context.Context, transactions []*rejectedTransaction) {
	for _, rejected := range transactions {
		s.pendingPool.remove(ctx, rejected.hash, rejected.status) // TODO(v1) make it a single call
	}
}
