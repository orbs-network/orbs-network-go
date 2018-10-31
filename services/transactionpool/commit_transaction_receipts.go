package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"time"
)

func (s *service) CommitTransactionReceipts(ctx context.Context, input *services.CommitTransactionReceiptsInput) (*services.CommitTransactionReceiptsOutput, error) {
	bh, _ := s.currentBlockHeightAndTime()
	if input.LastCommittedBlockHeight != bh+1 {
		return &services.CommitTransactionReceiptsOutput{
			NextDesiredBlockHeight:   bh + 1,
			LastCommittedBlockHeight: bh,
		}, nil
	}

	var myReceipts []*protocol.TransactionReceipt

	for _, receipt := range input.TransactionReceipts {
		removedTx := s.pendingPool.remove(ctx, receipt.Txhash(), protocol.TRANSACTION_STATUS_COMMITTED)
		if s.originatedFromMyPublicApi(removedTx) {
			myReceipts = append(myReceipts, receipt)
		}

		s.committedPool.add(receipt, timestampOrNow(removedTx))

		s.logger.Info("transaction receipt committed", log.String("flow", "checkpoint"), log.Stringable("txHash", receipt.Txhash()))

	}

	bh = s.updateBlockHeightAndTimestamp(input.ResultsBlockHeader)

	s.blockTracker.IncrementHeight()

	if len(myReceipts) > 0 {
		for _, handler := range s.transactionResultsHandlers {
			handler.HandleTransactionResults(ctx, &handlers.HandleTransactionResultsInput{
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

func (s *service) updateBlockHeightAndTimestamp(header *protocol.ResultsBlockHeader) primitives.BlockHeight {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mu.lastCommittedBlockHeight = header.BlockHeight()

	if header.Timestamp() == 0 {
		s.mu.lastCommittedBlockTimestamp = primitives.TimestampNano(time.Now().UnixNano()) //TODO remove this code when consensus-context actually sets block timestamp
		s.logger.Info("got 0 timestamp from results block header")
	} else {
		s.mu.lastCommittedBlockTimestamp = header.Timestamp()
	}

	return header.BlockHeight()
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
