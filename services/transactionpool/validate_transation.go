package transactionpool

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

func (s *service) validateTransaction(input *protocol.SignedTransaction) (bool, error) {
	if input.Transaction().InputArgumentsIterator().NextInputArguments().Uint64Value() > 1000 {
		//TODO: handle invalid transaction gracefully
		return false, nil
	}
	return true, nil
}
