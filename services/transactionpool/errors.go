package transactionpool

import (
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type ErrTransactionRejected struct {
	TransactionStatus protocol.TransactionStatus
}

func (e *ErrTransactionRejected) Error() string {
	return fmt.Sprintf("transaction rejected: %s", e.TransactionStatus)
}
