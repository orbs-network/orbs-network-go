package blockstorage

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
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

	// TODO(https://github.com/orbs-network/orbs-network-go/issues/524): the logic here aborting commits for already committed blocks is duplicated in the adapter because this is not under lock. synchronize to avoid duplicating logic in adapter
	if ok, err := s.validateBlockDoesNotExist(ctx, txBlockHeader, rsBlockHeader, lastCommittedBlock); err != nil || !ok {
		return nil, err
	}

	if err := s.validateBlockHeight(input.BlockPair, lastCommittedBlock); err != nil {
		return nil, err
	}

	if added, err := s.persistence.WriteNextBlock(input.BlockPair); err != nil || !added {
		return nil, err
	}

	s.metrics.blockHeight.Update(int64(input.BlockPair.TransactionsBlock.Header.BlockHeight()))

	s.nodeSync.HandleBlockCommitted(ctx)

	logger.Info("committed a block", log.BlockHeight(txBlockHeader.BlockHeight()), log.Int("num-transactions", len(input.BlockPair.TransactionsBlock.SignedTransactions)))

	return nil, nil
}
