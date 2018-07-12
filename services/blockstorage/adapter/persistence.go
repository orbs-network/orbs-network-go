package adapter

import "github.com/orbs-network/orbs-spec/types/go/protocol"

type BlockPersistence interface {
	WriteBlock(blockPairs *protocol.BlockPairContainer)
	ReadAllBlocks() []*protocol.BlockPairContainer
}
