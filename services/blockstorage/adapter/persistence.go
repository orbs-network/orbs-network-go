package adapter

import (
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type BlockPersistence interface {
	WithLogger(reporting instrumentation.BasicLogger) BlockPersistence
	WriteBlock(blockPairs *protocol.BlockPairContainer)
	ReadAllBlocks() []*protocol.BlockPairContainer
	GetTransactionsBlock(height primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error)
	GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error)
}
