package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type ContractState map[string]*protocol.StateRecord
type ChainState map[primitives.ContractName]ContractState

type StatePersistence interface {
	Write(height primitives.BlockHeight, ts primitives.TimestampNano, root primitives.MerkleSha256, diff ChainState) error
	Read(contract primitives.ContractName, key string) (*protocol.StateRecord, bool, error)
	ReadMetadata() (primitives.BlockHeight, primitives.TimestampNano, primitives.MerkleSha256, error)
	Each(callback func (contract primitives.ContractName, record *protocol.StateRecord)) error
}
