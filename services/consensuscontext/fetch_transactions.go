package consensuscontext

import (
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

func (s *service) fetchTransactions(maxNumberOfTransactions uint32,
	minimumTransactionsInBlock int, belowMinimalBlockDelayMillis uint32) (*services.GetTransactionsForOrderingOutput, error) {

	input := &services.GetTransactionsForOrderingInput{
		MaxNumberOfTransactions: maxNumberOfTransactions,
	}

	proposedTransactions, err := s.transactionPool.GetTransactionsForOrdering(input)
	if err != nil {
		return nil, err
	}
	txCount := len(proposedTransactions.SignedTransactions)
	if txCount >= minimumTransactionsInBlock {
		return proposedTransactions, nil
	}

	// TODO: Replace Sleep() with some other mechanism once we decide on it (such as context, timers...)
	time.Sleep(time.Duration(belowMinimalBlockDelayMillis) * time.Millisecond)

	proposedTransactions, err = s.transactionPool.GetTransactionsForOrdering(input)
	if err != nil {
		return nil, err
	}
	return proposedTransactions, nil

}
