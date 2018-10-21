package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type ContractDiff map[string]*protocol.StateRecord
type ChainDiff map[primitives.ContractName]ContractDiff

type StatePersistence interface {
	WriteState(height primitives.BlockHeight, ts primitives.TimestampNano, root primitives.MerkleSha256, diff ChainDiff) error
	ReadState(height primitives.BlockHeight, contract primitives.ContractName, key string) (*protocol.StateRecord, bool, error)
	ReadBlockHeight() (primitives.BlockHeight, error)
	ReadBlockTimestamp() (primitives.TimestampNano, error)
	ReadMerkleRoot(height primitives.BlockHeight) (primitives.MerkleSha256, error)
}
