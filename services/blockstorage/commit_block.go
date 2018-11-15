package blockstorage

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func (s *service) CommitBlock(ctx context.Context, input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	txBlockHeader := input.BlockPair.TransactionsBlock.Header
	rsBlockHeader := input.BlockPair.ResultsBlock.Header

	logger.Info("Trying to commit a block", log.BlockHeight(txBlockHeader.BlockHeight()))

	if err := s.validateProtocolVersion(input.BlockPair); err != nil {
		return nil, err
	}

	// the source of truth for the last committed block is persistence
	lastCommittedBlock, err := s.persistence.GetLastBlock()
	if err != nil {
		return nil, err
	}

	if ok, err := s.validateBlockDoesNotExist(ctx, txBlockHeader, rsBlockHeader, lastCommittedBlock); err != nil || !ok {
		return nil, err
	}

	if err := s.validateBlockHeight(input.BlockPair, lastCommittedBlock); err != nil {
		return nil, err
	}

	if err := s.persistence.WriteNextBlock(input.BlockPair); err != nil {
		return nil, err
	}

	s.metrics.blockHeight.Update(int64(input.BlockPair.TransactionsBlock.Header.BlockHeight()))

	s.blockSync.HandleBlockCommitted(ctx)

	logger.Info("committed a block", log.BlockHeight(txBlockHeader.BlockHeight()))

	if err := s.syncBlockToStateStorage(ctx, input.BlockPair); err != nil {
		// TODO: since the intra-node sync flow is self healing, we should not fail the entire commit if state storage is slow to sync
		s.logger.Error("intra-node sync to state storage failed", log.Error(err))
	}

	if err := s.syncBlockToTxPool(ctx, input.BlockPair); err != nil {
		// TODO: since the intra-node sync flow is self healing, should we fail if pool fails ?
		s.logger.Error("intra-node sync to tx pool failed", log.Error(err))
	}

	return nil, nil
} // TODO: this should not be called directly from CommitBlock, it should be called from a long living goroutine that continuously syncs the state storage

func (s *service) syncBlockToStateStorage(ctx context.Context, committedBlockPair *protocol.BlockPairContainer) error {
	_, err := s.stateStorage.CommitStateDiff(ctx, &services.CommitStateDiffInput{
		ResultsBlockHeader: committedBlockPair.ResultsBlock.Header,
		ContractStateDiffs: committedBlockPair.ResultsBlock.ContractStateDiffs,
	})
	return err
}

// TODO: this should not be called directly from CommitBlock, it should be called from a long living goroutine that continuously syncs the state storage
func (s *service) syncBlockToTxPool(ctx context.Context, committedBlockPair *protocol.BlockPairContainer) error {
	_, err := s.txPool.CommitTransactionReceipts(ctx, &services.CommitTransactionReceiptsInput{
		ResultsBlockHeader:       committedBlockPair.ResultsBlock.Header,
		TransactionReceipts:      committedBlockPair.ResultsBlock.TransactionReceipts,
		LastCommittedBlockHeight: committedBlockPair.ResultsBlock.Header.BlockHeight(),
	})
	return err
}
