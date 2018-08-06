package consensuscontext

import (
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
	"fmt"
)

func (s *service) fetchTransactions(input *services.GetTransactionsForOrderingInput, minimumTransactionsInBlock int, belowMinimalBlockDelayMillis uint32) (*services.GetTransactionsForOrderingOutput, error) {

	fmt.Println("Fetch transactions")
	var proposedTransactions *services.GetTransactionsForOrderingOutput = nil
	proposedTransactions, err := s.transactionPool.GetTransactionsForOrdering(input)
	if err != nil {
		return nil, err
	}
	txCount := len(proposedTransactions.SignedTransactions)
	if txCount >= minimumTransactionsInBlock {
		return proposedTransactions, nil
	}
	// TODO: How to test that Sleep() was called
	time.Sleep(time.Duration(belowMinimalBlockDelayMillis) * time.Millisecond)
	proposedTransactions, err = s.transactionPool.GetTransactionsForOrdering(input)
	if err != nil {
		return nil, err
	}
	return proposedTransactions, nil

}
