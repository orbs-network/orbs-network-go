package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"time"
)

func (s *service) CommitTransactionReceipts(input *services.CommitTransactionReceiptsInput) (*services.CommitTransactionReceiptsOutput, error) {
	bh, _ := s.currentBlockHeightAndTime()
	if input.LastCommittedBlockHeight != bh+1 {
		return &services.CommitTransactionReceiptsOutput{
			NextDesiredBlockHeight:   bh + 1,
			LastCommittedBlockHeight: bh,
		}, nil
	}

	var myReceipts []*protocol.TransactionReceipt

	for _, receipt := range input.TransactionReceipts {
		removedTx := s.pendingPool.remove(receipt.Txhash(), protocol.TRANSACTION_STATUS_COMMITTED)
		if s.originatedFromMyPublicApi(removedTx) {
			myReceipts = append(myReceipts, receipt)
		}

		s.committedPool.add(receipt, timestampOrNow(removedTx))

		s.logger.Info("transaction receipt committed", log.String("flow", "checkpoint"), log.Stringable("txHash", receipt.Txhash()))

	}

	s.mu.Lock()
	bh = input.ResultsBlockHeader.BlockHeight()
	bts := input.ResultsBlockHeader.Timestamp()
	s.mu.lastCommittedBlockHeight = bh
	s.mu.lastCommittedBlockTimestamp = bts
	if s.mu.lastCommittedBlockTimestamp == 0 {
		s.mu.lastCommittedBlockTimestamp = primitives.TimestampNano(time.Now().UnixNano()) //TODO remove this code when consensus-context actually sets block timestamp
		s.logger.Error("got 0 timestamp from results block header")
	}
	s.mu.Unlock()

	s.blockTracker.IncrementHeight()

	if len(myReceipts) > 0 {
		for _, handler := range s.transactionResultsHandlers {
			handler.HandleTransactionResults(&handlers.HandleTransactionResultsInput{
				BlockHeight:         bh,
				Timestamp:           input.ResultsBlockHeader.Timestamp(),
				TransactionReceipts: myReceipts,
			})
		}
	}

	s.logger.Info("committed transaction receipts for block height", log.BlockHeight(bh))

	return &services.CommitTransactionReceiptsOutput{
		NextDesiredBlockHeight:   bh + 1,
		LastCommittedBlockHeight: bh,
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
