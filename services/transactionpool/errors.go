package transactionpool

import (
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type ErrInvalidTransaction struct {
	TransactionStatus protocol.TransactionStatus
}

func (e *ErrInvalidTransaction) Error() string {
	return fmt.Sprintf("Invalid Transaction: %v", e.TransactionStatus)
}
