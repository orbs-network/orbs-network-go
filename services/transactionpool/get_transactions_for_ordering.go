package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func (s *service) GetTransactionsForOrdering(ctx context.Context, input *services.GetTransactionsForOrderingInput) (*services.GetTransactionsForOrderingOutput, error) {

	if err := s.blockTracker.WaitForBlock(input.BlockHeight); err != nil {
		return nil, err
	}

	out := &services.GetTransactionsForOrderingOutput{}
	transactions := s.pendingPool.getBatch(input.MaxNumberOfTransactions, input.MaxTransactionsSetSizeKb*1024)
	vctx := s.createValidationContext()

	transactionsForPreOrder := make(Transactions, 0, input.MaxNumberOfTransactions)
	for _, tx := range transactions {
		txHash := digest.CalcTxHash(tx.Transaction())
		if err := vctx.validateTransaction(tx); err != nil {
			s.logger.Info("dropping invalid transaction", log.Error(err), log.String("flow", "checkpoint"), log.Stringable("txHash", txHash))
			s.pendingPool.remove(txHash, err.TransactionStatus)
		} else if alreadyCommitted := s.committedPool.get(txHash); alreadyCommitted != nil {
			s.logger.Info("dropping committed transaction", log.String("flow", "checkpoint"), log.Stringable("txHash", txHash))
			s.pendingPool.remove(txHash, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED)

		} else {
			transactionsForPreOrder = append(transactionsForPreOrder, tx)
		}
	}

	//TODO handle error from vm
	bh, _ := s.currentBlockHeightAndTime()
	preOrderResults, _ := s.virtualMachine.TransactionSetPreOrder(&services.TransactionSetPreOrderInput{
		SignedTransactions: transactionsForPreOrder,
		BlockHeight:        bh,
	})

	for i := range transactionsForPreOrder {
		tx := transactionsForPreOrder[i]
		if preOrderResults.PreOrderResults[i] == protocol.TRANSACTION_STATUS_PRE_ORDER_VALID {
			out.SignedTransactions = append(out.SignedTransactions, tx)
		} else {
			txHash := digest.CalcTxHash(tx.Transaction()) //TODO we calculate TX hash again even though we calculated it above while iterating. Consider memoization.
			s.logger.Info("dropping transaction that failed pre-order validation", log.String("flow", "checkpoint"), log.Stringable("txHash", txHash))
			s.pendingPool.remove(txHash, protocol.TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER)
		}
	}

	return out, nil
}
