package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type ContractDiff map[string]*protocol.StateRecord
type ChainDiff map[primitives.ContractName]ContractDiff

type StatePersistence interface {
	Write(height primitives.BlockHeight, ts primitives.TimestampNano, root primitives.MerkleSha256, diff ChainDiff) error
	Read(contract primitives.ContractName, key string) (*protocol.StateRecord, bool, error)
	ReadMetadata() (primitives.BlockHeight, primitives.TimestampNano, primitives.MerkleSha256, error)
}
