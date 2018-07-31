package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type BlockPersistence interface {
	WriteBlock(blockPairs *protocol.BlockPairContainer)
	ReadAllBlocks() []*protocol.BlockPairContainer
	GetTransactionsBlock(height primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error)
	GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error)
}
