package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type Config interface {
	NodeId() string
}

type blockPersistence struct {
	blockWritten chan bool
	blockPairs []protocol.BlockPair
	config       Config
}

func NewBlockPersistence(config Config) BlockPersistence {
	return &blockPersistence{
		config:         config,
		blockWritten: make(chan bool, 10),
	}
}

func (bp *blockPersistence) WriteBlock(blockPair *protocol.BlockPair) {
	bp.blockPairs = append(bp.blockPairs, *blockPair)
	bp.blockWritten <- true
}

func (bp *blockPersistence) ReadAllBlocks() []protocol.BlockPair {
	return bp.blockPairs
}