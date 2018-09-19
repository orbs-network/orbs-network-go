package builders

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
)

func ClientCallMethodResponseOutputArgumentsDecode(r *client.CallMethodResponse) *protocol.MethodArgumentArrayArgumentsIterator {
	argsArray := protocol.MethodArgumentArrayReader(r.RawOutputArgumentArrayWithHeader())
	return argsArray.ArgumentsIterator()
}

// encode MethodArgumentsOpaque with MethodArgumentsOpaqueEncode

func ClientCallMethodResponseOutputArgumentsPrint(r *client.CallMethodResponse) string {
	argsArray := protocol.MethodArgumentArrayReader(r.RawOutputArgumentArrayWithHeader())
	return argsArray.String()
}
