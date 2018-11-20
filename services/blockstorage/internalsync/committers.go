package internalsync

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type stateStorageCommitter struct {
	service services.StateStorage
}

type transactionPoolCommitter struct {
	service services.TransactionPool
}

func NewTxPoolCommitter(txPool services.TransactionPool) *transactionPoolCommitter {
	return &transactionPoolCommitter{service: txPool}
}

func NewStateStorageCommitter(stateStorage services.StateStorage) *stateStorageCommitter {
	return &stateStorageCommitter{service: stateStorage}
}

func (ssc *stateStorageCommitter) blockSyncFunc(ctx context.Context, committedBlockPair *protocol.BlockPairContainer) (primitives.BlockHeight, error) {
	out, err := ssc.service.CommitStateDiff(ctx, &services.CommitStateDiffInput{
		ResultsBlockHeader: committedBlockPair.ResultsBlock.Header,
		ContractStateDiffs: committedBlockPair.ResultsBlock.ContractStateDiffs,
	})
	return out.NextDesiredBlockHeight, err
}

func (tpc transactionPoolCommitter) blockSyncFunc(ctx context.Context, committedBlockPair *protocol.BlockPairContainer) (primitives.BlockHeight, error) {
	out, err := tpc.service.CommitTransactionReceipts(ctx, &services.CommitTransactionReceiptsInput{
		ResultsBlockHeader:       committedBlockPair.ResultsBlock.Header,
		TransactionReceipts:      committedBlockPair.ResultsBlock.TransactionReceipts,
		LastCommittedBlockHeight: committedBlockPair.ResultsBlock.Header.BlockHeight(),
	})
	return out.NextDesiredBlockHeight, err
}