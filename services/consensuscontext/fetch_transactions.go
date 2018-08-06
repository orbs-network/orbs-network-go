package consensuscontext

import (
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
	"fmt"
)

func (s *service) fetchTransactions(input *services.GetTransactionsForOrderingInput, maxAttempts int, intervalBetweenAttempts time.Duration) (*services.GetTransactionsForOrderingOutput, error) {

	fmt.Println("Fetch transactions")
	var proposedTransactions *services.GetTransactionsForOrderingOutput = nil
	for i := 0; i < maxAttempts; i++ {
		proposedTransactions, err := s.transactionPool.GetTransactionsForOrdering(input)
		if err != nil {
			return nil, err
		}
		txCount := len(proposedTransactions.SignedTransactions)
		if txCount > 0 {
			return proposedTransactions, nil
		}
		// TODO: How to test that Sleep() was called
		fmt.Println("Before sleep")
		time.Sleep(intervalBetweenAttempts)
		fmt.Println("After sleep")
	}
	return proposedTransactions, nil

}
