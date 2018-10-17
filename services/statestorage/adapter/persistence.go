package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type StatePersistence interface {
	WriteState(height primitives.BlockHeight, contractStateDiffs []*protocol.ContractStateDiff) error
	ReadState(height primitives.BlockHeight, contract primitives.ContractName, key string) (*protocol.StateRecord, bool, error)
	WriteMerkleRoot(height primitives.BlockHeight, sha256 primitives.MerkleSha256) error
	ReadMerkleRoot(height primitives.BlockHeight) (primitives.MerkleSha256, error)
}
