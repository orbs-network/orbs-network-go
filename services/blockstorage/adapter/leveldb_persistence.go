package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type levelDbBlockPersistence struct {
	blockWritten chan bool
	blockPairs   []*protocol.BlockPairContainer
}

func NewLevelDbBlockPersistence() BlockPersistence {
	return &levelDbBlockPersistence{
		blockWritten: make(chan bool, 10),
	}
}

func (bp *levelDbBlockPersistence) WriteBlock(blockPair *protocol.BlockPairContainer) error {
	bp.blockPairs = append(bp.blockPairs, blockPair)
	//bp.blockWritten <- true

	return nil
}

func (bp *levelDbBlockPersistence) ReadAllBlocks() []*protocol.BlockPairContainer {
	return bp.blockPairs
}

func (bp *levelDbBlockPersistence) GetReceiptRelevantBlocks(txTimeStamp primitives.TimestampNano, rules BlockSearchRules) []*protocol.BlockPairContainer {
	panic("not implemented")
}

func (bp *levelDbBlockPersistence) GetTransactionsBlock(height primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error) {
	panic("not implemented")
}

func (bp *levelDbBlockPersistence) GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error) {
	panic("not implemented")
}
