package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func (s *service) fetchTransactions(ctx context.Context, currentBlockHeight primitives.BlockHeight, prevBlockTimestamp primitives.TimestampNano, maxNumberOfTransactions uint32) (*services.GetTransactionsForOrderingOutput, error) {

	input := &services.GetTransactionsForOrderingInput{
		MaxTransactionsSetSizeKb: 0, // TODO(v1): either fill in or delete from spec
		MaxNumberOfTransactions:  maxNumberOfTransactions,
		CurrentBlockHeight:       currentBlockHeight,
		PrevBlockTimestamp:       prevBlockTimestamp,
	}

	proposedTransactions, err := s.transactionPool.GetTransactionsForOrdering(ctx, input)
	if err != nil {
		return nil, err
	}

	return proposedTransactions, nil
}
