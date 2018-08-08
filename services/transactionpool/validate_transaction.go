package transactionpool

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"time"
)

func validateTransaction(transaction *protocol.SignedTransaction) error {
	return validateTransactionTimestamp(transaction)
}

func validateTransactionTimestamp(transaction *protocol.SignedTransaction) error {
	if uint64(transaction.Transaction().Timestamp()) > uint64(time.Now().UnixNano()) {
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_TIME_STAMP_WINDOW_EXCEEDED}
	}
	return nil
}
