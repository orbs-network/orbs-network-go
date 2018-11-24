package adapter

import (
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type BlockSearchRules struct {
	StartGraceNano        int64
	EndGraceNano          int64
	TransactionExpireNano int64
}

type BlockPersistence interface {
	WriteNextBlock(blockPairs *protocol.BlockPairContainer) (bool, error)
	GetLastBlock() (*protocol.BlockPairContainer, error)

	// TODO: this function has a hideous interface
	GetBlocks(first primitives.BlockHeight, last primitives.BlockHeight) (blocks []*protocol.BlockPairContainer, firstReturnedBlockHeight primitives.BlockHeight, lastReturnedBlockHeight primitives.BlockHeight, err error)
	GetNumBlocks() (primitives.BlockHeight, error)

	GetBlockTracker() *synchronization.BlockTracker
	GetTransactionsBlock(height primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error)
	GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error)

	// TODO: kill this function and move its logic into the adapter
	GetBlocksRelevantToTxTimestamp(txTimeStamp primitives.TimestampNano, rules BlockSearchRules) []*protocol.BlockPairContainer
}
