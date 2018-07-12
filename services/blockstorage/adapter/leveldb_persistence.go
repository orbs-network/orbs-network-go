package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type Config interface {
	NodeId() string
}

type levelDbBlockPersistence struct {
	blockWritten chan bool
	blockPairs   []*protocol.BlockPairContainer
	config       Config
}

func NewLevelDbBlockPersistence(config Config) BlockPersistence {
	return &levelDbBlockPersistence{
		config:       config,
		blockWritten: make(chan bool, 10),
	}
}

func (bp *levelDbBlockPersistence) WriteBlock(blockPair *protocol.BlockPairContainer) {
	bp.blockPairs = append(bp.blockPairs, blockPair)
	bp.blockWritten <- true
}

func (bp *levelDbBlockPersistence) ReadAllBlocks() []*protocol.BlockPairContainer {
	return bp.blockPairs
}
