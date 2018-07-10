package adapter

import "github.com/orbs-network/orbs-spec/types/go/protocol"

type BlockPersistence interface {
	WriteBlock(signedTransaction *protocol.SignedTransaction)
	ReadAllBlocks() []protocol.SignedTransaction
}
