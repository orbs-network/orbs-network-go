package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

func (s *service) fetchTransactions(ctx context.Context, currentBlockHeight primitives.BlockHeight, currentBlockTimestamp primitives.TimestampNano, maxNumberOfTransactions uint32,
	minimumTransactionsInBlock uint32, minimalBlockDelay time.Duration) (*services.GetTransactionsForOrderingOutput, error) {

	input := &services.GetTransactionsForOrderingInput{
		MaxTransactionsSetSizeKb: 0, // TODO(v1): either fill in or delete from spec
		MaxNumberOfTransactions:  maxNumberOfTransactions,
		CurrentBlockHeight:       currentBlockHeight,
		CurrentBlockTimestamp:    currentBlockTimestamp,
	}

	proposedTransactions, err := s.transactionPool.GetTransactionsForOrdering(ctx, input)
	if err != nil {
		return nil, err
	}
	txCount := uint32(len(proposedTransactions.SignedTransactions))
	if txCount >= minimumTransactionsInBlock {
		return proposedTransactions, nil
	}

	// FIXME v1 find better way to wait for new block, maybe block tracker?
	<-time.After(minimalBlockDelay)

	proposedTransactions, err = s.transactionPool.GetTransactionsForOrdering(ctx, input)
	if err != nil {
		return nil, err
	}
	return proposedTransactions, nil

}
