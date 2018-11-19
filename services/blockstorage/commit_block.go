package blockstorage

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
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

	s.extSync.HandleBlockCommitted(ctx)

	logger.Info("committed a block", log.BlockHeight(txBlockHeader.BlockHeight()))

	return nil, nil
}

func (s *service) syncBlockToStateStorage(ctx context.Context, committedBlockPair *protocol.BlockPairContainer) (primitives.BlockHeight, error) {
	out, err := s.stateStorage.CommitStateDiff(ctx, &services.CommitStateDiffInput{
		ResultsBlockHeader: committedBlockPair.ResultsBlock.Header,
		ContractStateDiffs: committedBlockPair.ResultsBlock.ContractStateDiffs,
	})
	return out.NextDesiredBlockHeight, err
}

func (s *service) syncBlockToTxPool(ctx context.Context, committedBlockPair *protocol.BlockPairContainer) (primitives.BlockHeight, error) {
	out, err := s.txPool.CommitTransactionReceipts(ctx, &services.CommitTransactionReceiptsInput{
		ResultsBlockHeader:       committedBlockPair.ResultsBlock.Header,
		TransactionReceipts:      committedBlockPair.ResultsBlock.TransactionReceipts,
		LastCommittedBlockHeight: committedBlockPair.ResultsBlock.Header.BlockHeight(),
	})
	return out.NextDesiredBlockHeight, err
}
