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
	WriteBlock(blockPairs *protocol.BlockPairContainer) error
	// FIXME kill it
	ReadAllBlocks() []*protocol.BlockPairContainer
	GetBlocks(first primitives.BlockHeight, last primitives.BlockHeight) ([]*protocol.BlockPairContainer, error)
	GetLastBlock() (*protocol.BlockPairContainer, error)
	GetBlockTracker() *synchronization.BlockTracker
	GetReceiptRelevantBlocks(txTimeStamp primitives.TimestampNano, rules BlockSearchRules) []*protocol.BlockPairContainer
	GetTransactionsBlock(height primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error)
	GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error)
}
