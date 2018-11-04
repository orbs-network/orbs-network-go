package transactionpool

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) ValidateTransactionsForOrdering(ctx context.Context, input *services.ValidateTransactionsForOrderingInput) (*services.ValidateTransactionsForOrderingOutput, error) {
	timeoutCtx, _ := context.WithTimeout(ctx, s.config.BlockTrackerGraceTimeout())
	if err := s.blockTracker.WaitForBlock(timeoutCtx, input.BlockHeight); err != nil {
		return nil, err
	}

	vctx := s.createValidationContext()

	for _, tx := range input.SignedTransactions {
		txHash := digest.CalcTxHash(tx.Transaction())
		if s.committedPool.has(txHash) {
			return nil, errors.Errorf("transaction with hash %s already committed", txHash)
		}

		if err := vctx.validateTransaction(tx); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("transaction with hash %s is invalid", txHash))
		}
	}

	//TODO handle error from vm
	bh, _ := s.currentBlockHeightAndTime()
	preOrderResults, _ := s.virtualMachine.TransactionSetPreOrder(ctx, &services.TransactionSetPreOrderInput{
		SignedTransactions: input.SignedTransactions,
		BlockHeight:        bh,
	})

	for i, tx := range input.SignedTransactions {
		if status := preOrderResults.PreOrderResults[i]; status != protocol.TRANSACTION_STATUS_PRE_ORDER_VALID {
			return nil, errors.Errorf("transaction with hash %s failed pre-order checks with status %s", digest.CalcTxHash(tx.Transaction()), status)
		}
	}
	return &services.ValidateTransactionsForOrderingOutput{}, nil
}
