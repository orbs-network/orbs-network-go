package adapter

import "github.com/orbs-network/orbs-spec/types/go/protocol"

type StatePersistence interface {
	WriteState(stateDiffs *protocol.StateRecord)
	ReadState() []protocol.StateRecord
}
