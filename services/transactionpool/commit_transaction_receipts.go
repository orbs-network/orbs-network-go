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

	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	if bh, _ := s.lastCommittedBlockHeightAndTime(); input.LastCommittedBlockHeight != bh+1 {
		return &services.CommitTransactionReceiptsOutput{
			NextDesiredBlockHeight:   bh + 1,
			LastCommittedBlockHeight: bh,
		}, nil
	}

	s.addCommitLock.Lock()
	defer s.addCommitLock.Unlock()

	newBh, ts := s.updateBlockHeightAndTimestamp(input.ResultsBlockHeader) //TODO(v1): should this be updated separately from blockTracker? are we updating block height too early?

	c := &committer{logger: logger, adder: s.committedPool, remover: s.pendingPool, nodeAddress: s.config.NodeAddress(), blockHeight: newBh, blockTime: ts}

	c.commit(ctx, input.TransactionReceipts...)

	s.blockTracker.IncrementTo(newBh)
	s.blockHeightReporter.IncrementTo(newBh)

	c.notify(ctx, s.transactionResultsHandlers...)

	logger.Info("committed transaction receipts for block height", log.BlockHeight(newBh))

	return &services.CommitTransactionReceiptsOutput{
		NextDesiredBlockHeight:   newBh + 1,
		LastCommittedBlockHeight: newBh,
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

type adder interface {
	add(receipt *protocol.TransactionReceipt, submitted primitives.TimestampNano, blockHeight primitives.BlockHeight, blockTs primitives.TimestampNano)
}

type remover interface {
	remove(ctx context.Context, txHash primitives.Sha256, removalReason protocol.TransactionStatus) *primitives.NodeAddress
}

type committer struct {
	adder       adder
	remover     remover
	nodeAddress primitives.NodeAddress
	logger      log.BasicLogger
	blockHeight primitives.BlockHeight
	blockTime   primitives.TimestampNano

	myReceipts []*protocol.TransactionReceipt
}

func (c *committer) commit(ctx context.Context, receipts ...*protocol.TransactionReceipt) (myReceipts []*protocol.TransactionReceipt) {

	for _, receipt := range receipts {
		c.adder.add(receipt, c.blockTime, c.blockHeight, c.blockTime) // tx MUST be added to committed pool prior to removing it from pending pool, otherwise the same tx can be added again, since we do not remove and add atomically
		removedTxGateway := c.remover.remove(ctx, receipt.Txhash(), protocol.TRANSACTION_STATUS_COMMITTED)
		if c.amITheGatewayOf(removedTxGateway) {
			c.myReceipts = append(c.myReceipts, receipt)
		}

		c.logger.Info("transaction receipt committed", log.String("flow", "checkpoint"), log.Transaction(receipt.Txhash()))
	}

	return
}

func (c *committer) amITheGatewayOf(removedTxGateway *primitives.NodeAddress) bool {
	return removedTxGateway != nil && removedTxGateway.Equal(c.nodeAddress)
}

func (c *committer) notify(ctx context.Context, resultsHandlers ...handlers.TransactionResultsHandler) {

	if len(c.myReceipts) > 0 {
		for _, handler := range resultsHandlers {
			_, err := handler.HandleTransactionResults(ctx, &handlers.HandleTransactionResultsInput{
				BlockHeight:         c.blockHeight,
				Timestamp:           c.blockTime,
				TransactionReceipts: c.myReceipts,
			})
			if err != nil {
				c.logger.Info("notify tx result failed", log.Error(err))
			}
		}
	}
}
