package adapter

import (
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

// A Callback function receiving block pages. ScanBlocks() will call this function
// repeatedly until it returns false or there are no more blocks to fetch
type CursorFunc func(first primitives.BlockHeight, page []*protocol.BlockPairContainer) (wantsMore bool)

type BlockPersistence interface {
	WriteNextBlock(blockPairs *protocol.BlockPairContainer) (bool, error)

	ScanBlocks(from primitives.BlockHeight, pageSize uint8, f CursorFunc) error

	GetLastBlockHeight() (primitives.BlockHeight, error)
	GetLastBlock() (*protocol.BlockPairContainer, error)

	GetTransactionsBlock(height primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error)
	GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error)
	GetBlockByTx(txHash primitives.Sha256, minBlockTs primitives.TimestampNano, maxBlockTs primitives.TimestampNano) (block *protocol.BlockPairContainer, txIndexInBlock int, err error)

	GetBlockTracker() *synchronization.BlockTracker
}
