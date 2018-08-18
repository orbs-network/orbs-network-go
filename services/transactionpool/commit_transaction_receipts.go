package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"time"
)

func (s *service) CommitTransactionReceipts(input *services.CommitTransactionReceiptsInput) (*services.CommitTransactionReceiptsOutput, error) {
	if input.LastCommittedBlockHeight != s.lastCommittedBlockHeight+1 {
		return &services.CommitTransactionReceiptsOutput{
			NextDesiredBlockHeight:   s.lastCommittedBlockHeight + 1,
			LastCommittedBlockHeight: s.lastCommittedBlockHeight,
		}, nil
	}

	var myReceipts []*protocol.TransactionReceipt

	for _, receipt := range input.TransactionReceipts {
		removedTx := s.pendingPool.remove(receipt.Txhash())
		if s.originatedFromMyPublicApi(removedTx) {
			myReceipts = append(myReceipts, receipt)
		}

		s.committedPool.add(receipt, timestampOrNow(removedTx))

	}

	s.lastCommittedBlockHeight = input.ResultsBlockHeader.BlockHeight()
	s.lastCommittedBlockTimestamp = input.ResultsBlockHeader.Timestamp()

	s.blockTracker.IncrementHeight()

	for _, handler := range s.transactionResultsHandlers {
		handler.HandleTransactionResults(&handlers.HandleTransactionResultsInput{
			BlockHeight:         s.lastCommittedBlockHeight,
			Timestamp:           input.ResultsBlockHeader.Timestamp(),
			TransactionReceipts: myReceipts,
		})
	}

	s.log.Info("committed transaction receipts for block height", instrumentation.BlockHeight(s.lastCommittedBlockHeight))

	return &services.CommitTransactionReceiptsOutput{
		NextDesiredBlockHeight:   s.lastCommittedBlockHeight + 1,
		LastCommittedBlockHeight: s.lastCommittedBlockHeight,
	}, nil
}
func timestampOrNow(tx *pendingTransaction) primitives.TimestampNano {
	if tx != nil {
		return tx.transaction.Transaction().Timestamp()
	} else {
		return primitives.TimestampNano(time.Now().UnixNano())
	}
}

func (s *service) originatedFromMyPublicApi(removedTx *pendingTransaction) bool {
	return removedTx != nil && removedTx.gatewayPublicKey.Equal(s.config.NodePublicKey())
}
