package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	return &services.GetCommittedTransactionReceiptOutput{
		TransactionStatus:  status,
		TransactionReceipt: receipt,
		BlockHeight:        s.mu.lastCommittedBlockHeight,
		BlockTimestamp:     s.mu.lastCommittedBlockTimestamp,
	}
}

func (s *service) currentNodeTimeWithGrace() primitives.TimestampNano {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mu.lastCommittedBlockTimestamp + primitives.TimestampNano(s.config.TransactionPoolFutureTimestampGraceTimeout().Nanoseconds())
}
