package adapter

import (
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type BlockPersistence interface {
	WriteNextBlock(blockPairs *protocol.BlockPairContainer) (bool, error)
	GetLastBlock() (*protocol.BlockPairContainer, error)

	// TODO(v1): this function has a hideous interface
	GetBlocks(first primitives.BlockHeight, last primitives.BlockHeight) (blocks []*protocol.BlockPairContainer, firstReturnedBlockHeight primitives.BlockHeight, lastReturnedBlockHeight primitives.BlockHeight, err error)
	GetNumBlocks() (primitives.BlockHeight, error)

	GetBlockTracker() *synchronization.BlockTracker
	GetTransactionsBlock(height primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error)
	GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error)

	// TODO(v1): kill this function and move its logic into the adapter
	GetBlockByTx(txHash primitives.Sha256, minBlockTs primitives.TimestampNano, maxBlockTs primitives.TimestampNano) (*protocol.BlockPairContainer, int, error)
}
