package adapter

import (
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type BlockPersistence interface {
	WriteNextBlock(blockPairs *protocol.BlockPairContainer) (bool, error)

	ScanBlocks(from primitives.BlockHeight, pageSize uint, f func(offset primitives.BlockHeight, page []*protocol.BlockPairContainer) bool) error

	GetNumBlocks() (primitives.BlockHeight, error)

	GetBlockTracker() *synchronization.BlockTracker
	GetLastBlock() (*protocol.BlockPairContainer, error)
	GetTransactionsBlock(height primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error)
	GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error)

	GetBlockByTx(txHash primitives.Sha256, minBlockTs primitives.TimestampNano, maxBlockTs primitives.TimestampNano) (*protocol.BlockPairContainer, int, error)
}
