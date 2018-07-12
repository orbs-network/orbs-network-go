package adapter

import "github.com/orbs-network/orbs-spec/types/go/protocol"

type BlockPersistence interface {
	WriteBlock(blockPairs *protocol.BlockPair)
	ReadAllBlocks() []protocol.BlockPair
}
