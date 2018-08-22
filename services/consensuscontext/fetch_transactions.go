package consensuscontext

import (
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

func (s *service) fetchTransactions(maxNumberOfTransactions uint32,
	minimumTransactionsInBlock uint32, belowMinimalBlockDelayMillis time.Duration) (*services.GetTransactionsForOrderingOutput, error) {

	input := &services.GetTransactionsForOrderingInput{
		MaxNumberOfTransactions: maxNumberOfTransactions,
	}

	proposedTransactions, err := s.transactionPool.GetTransactionsForOrdering(input)
	if err != nil {
		return nil, err
	}
	txCount := uint32(len(proposedTransactions.SignedTransactions))
	if txCount >= minimumTransactionsInBlock {
		return proposedTransactions, nil
	}

	// TODO should we wait here at all?
	<-time.After(belowMinimalBlockDelayMillis)

	proposedTransactions, err = s.transactionPool.GetTransactionsForOrdering(input)
	if err != nil {
		return nil, err
	}
	return proposedTransactions, nil

}
