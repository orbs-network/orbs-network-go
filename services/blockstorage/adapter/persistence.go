package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type BlockPersistence interface {
	WriteBlock(blockPairs *protocol.BlockPairContainer) error
	ReadAllBlocks() []*protocol.BlockPairContainer
	ReadAllBlocksByTimeRange(start, end primitives.TimestampNano) []*protocol.BlockPairContainer
	GetTransactionsBlock(height primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error)
	GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error)
}
