package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func (s *service) GetCommittedTransactionReceipt(ctx context.Context, input *services.GetCommittedTransactionReceiptInput) (*services.GetCommittedTransactionReceiptOutput, error) {

	if tx := s.pendingPool.get(input.Txhash); tx != nil {
		return s.getTxResult(nil, protocol.TRANSACTION_STATUS_PENDING), nil
	}

	if tx := s.committedPool.get(input.Txhash); tx != nil {
		return &services.GetCommittedTransactionReceiptOutput{
			TransactionStatus:  protocol.TRANSACTION_STATUS_COMMITTED,
			TransactionReceipt: tx.receipt,
			BlockHeight:        tx.blockHeight,
			BlockTimestamp:     tx.blockTimestamp,
		}, nil
	}

	return s.getTxResult(nil, protocol.TRANSACTION_STATUS_NO_RECORD_FOUND), nil
}

func (s *service) getTxResult(receipt *protocol.TransactionReceipt, status protocol.TransactionStatus) *services.GetCommittedTransactionReceiptOutput {
	s.lastCommitted.RLock()
	defer s.lastCommitted.RUnlock()
	return &services.GetCommittedTransactionReceiptOutput{
		TransactionStatus:  status,
		TransactionReceipt: receipt,
		BlockHeight:        s.lastCommitted.blockHeight,
		BlockTimestamp:     s.lastCommitted.timestamp,
	}
}
