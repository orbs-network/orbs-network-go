package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

func (s *service) CommitTransactionReceipts(ctx context.Context, input *services.CommitTransactionReceiptsInput) (*services.CommitTransactionReceiptsOutput, error) {
	s.addCommitLock.Lock()
	defer s.addCommitLock.Unlock()

	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	if bh, _ := s.lastCommittedBlockHeightAndTime(); input.LastCommittedBlockHeight != bh+1 {
		return &services.CommitTransactionReceiptsOutput{
			NextDesiredBlockHeight:   bh + 1,
			LastCommittedBlockHeight: bh,
		}, nil
	}

	bh, ts := s.updateBlockHeightAndTimestamp(input.ResultsBlockHeader) //TODO(v1): should this be updated separately from blockTracker? are we updating block height too early?

	var myReceipts []*protocol.TransactionReceipt

	for _, receipt := range input.TransactionReceipts {
		s.committedPool.add(receipt, ts, bh, ts) // tx MUST be added to committed pool prior to removing it from pending pool, otherwise the same tx can be added again, since we do not remove and add atomically
		removedTx := s.pendingPool.remove(ctx, receipt.Txhash(), protocol.TRANSACTION_STATUS_COMMITTED)
		if s.originatedFromMyPublicApi(removedTx) {
			myReceipts = append(myReceipts, receipt)
		}

		logger.Info("transaction receipt committed", log.String("flow", "checkpoint"), log.Transaction(receipt.Txhash()))

	}

	s.blockTracker.IncrementTo(bh)
	s.blockHeightReporter.IncrementTo(bh)

	if len(myReceipts) > 0 {
		for _, handler := range s.transactionResultsHandlers {
			_, err := handler.HandleTransactionResults(ctx, &handlers.HandleTransactionResultsInput{
				BlockHeight:         bh,
				Timestamp:           input.ResultsBlockHeader.Timestamp(),
				TransactionReceipts: myReceipts,
			})
			if err != nil {
				logger.Info("notify tx result failed", log.Error(err))
			}
		}
	}

	logger.Info("committed transaction receipts for block height", log.BlockHeight(bh))

	return &services.CommitTransactionReceiptsOutput{
		NextDesiredBlockHeight:   bh + 1,
		LastCommittedBlockHeight: bh,
	}, nil
}

func (s *service) updateBlockHeightAndTimestamp(header *protocol.ResultsBlockHeader) (primitives.BlockHeight, primitives.TimestampNano) {
	s.lastCommitted.Lock()
	defer s.lastCommitted.Unlock()

	s.lastCommitted.blockHeight = header.BlockHeight()
	s.lastCommitted.timestamp = header.Timestamp()
	s.metrics.blockHeight.Update(int64(header.BlockHeight()))

	s.logger.Info("transaction pool reached block height", log.BlockHeight(header.BlockHeight()))

	return header.BlockHeight(), header.Timestamp()
}

func (s *service) originatedFromMyPublicApi(removedTx *pendingTransaction) bool {
	return removedTx != nil && removedTx.gatewayNodeAddress.Equal(s.config.NodeAddress())
}
