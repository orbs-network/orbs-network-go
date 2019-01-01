package builders

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
)

func ClientCallMethodResponseOutputArgumentsDecode(r *client.CallMethodResponse) *protocol.ArgumentArrayArgumentsIterator {
	argsArray := protocol.ArgumentArrayReader(r.RawOutputArgumentArrayWithHeader())
	return argsArray.ArgumentsIterator()
}

// encode ArgumentsOpaque with PackedArgumentArrayEncode

func ClientCallMethodResponseOutputArgumentsPrint(r *client.CallMethodResponse) string {
	argsArray := protocol.ArgumentArrayReader(r.RawOutputArgumentArrayWithHeader())
	return argsArray.String()
}
