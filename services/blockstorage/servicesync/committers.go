package servicesync

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type serviceDesc struct {
	name string
}
type stateStorageCommitter struct {
	serviceDesc
	service services.StateStorage
}

type transactionPoolCommitter struct {
	serviceDesc
	service services.TransactionPool
}

func NewTxPoolCommitter(txPool services.TransactionPool) *transactionPoolCommitter {
	return &transactionPoolCommitter{service: txPool, serviceDesc: serviceDesc{"tx-pool-sync"}}
}

func NewStateStorageCommitter(stateStorage services.StateStorage) *stateStorageCommitter {
	return &stateStorageCommitter{service: stateStorage, serviceDesc: serviceDesc{"state-storage-sync"}}
}

func (ssc *stateStorageCommitter) commitBlockPair(ctx context.Context, committedBlockPair *protocol.BlockPairContainer) (primitives.BlockHeight, error) {
	out, err := ssc.service.CommitStateDiff(ctx, &services.CommitStateDiffInput{
		ResultsBlockHeader: committedBlockPair.ResultsBlock.Header,
		ContractStateDiffs: committedBlockPair.ResultsBlock.ContractStateDiffs,
	})
	return out.NextDesiredBlockHeight, err
}

func (tpc *transactionPoolCommitter) commitBlockPair(ctx context.Context, committedBlockPair *protocol.BlockPairContainer) (primitives.BlockHeight, error) {
	out, err := tpc.service.CommitTransactionReceipts(ctx, &services.CommitTransactionReceiptsInput{
		ResultsBlockHeader:       committedBlockPair.ResultsBlock.Header,
		TransactionReceipts:      committedBlockPair.ResultsBlock.TransactionReceipts,
		LastCommittedBlockHeight: committedBlockPair.ResultsBlock.Header.BlockHeight(),
	})
	return out.NextDesiredBlockHeight, err
}

func (sd *serviceDesc) getServiceName() string {
	return sd.name
}
