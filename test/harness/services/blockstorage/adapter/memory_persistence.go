package adapter

import (
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type InMemoryBlockPersistence interface {
	adapter.BlockPersistence
	WaitForBlocks(count int)
}

type inMemoryBlockPersistence struct {
	blockWritten chan bool
	blockPairs   []*protocol.BlockPairContainer
	config       adapter.Config
}

func NewInMemoryBlockPersistence(config adapter.Config) InMemoryBlockPersistence {
	return &inMemoryBlockPersistence{
		config:       config,
		blockWritten: make(chan bool, 10),
	}
}

func (bp *inMemoryBlockPersistence) WaitForBlocks(count int) {
	for i := 0; i < count; i++ {
		<-bp.blockWritten
	}
}

func (bp *inMemoryBlockPersistence) WriteBlock(blockPair *protocol.BlockPairContainer) {
	bp.blockPairs = append(bp.blockPairs, blockPair)
	bp.blockWritten <- true
}

func (bp *inMemoryBlockPersistence) ReadAllBlocks() []*protocol.BlockPairContainer {
	return bp.blockPairs
}
