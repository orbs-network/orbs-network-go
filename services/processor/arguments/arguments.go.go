package arguments

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"math/big"
)

func ArgsToArgumentArray(args ...interface{}) *protocol.ArgumentArray {
	res := []*protocol.ArgumentBuilder{}
	for _, arg := range args {
		switch arg.(type) {
		case uint32:
			res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_UINT_32_VALUE, Uint32Value: arg.(uint32)})
		case uint64:
			res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: arg.(uint64)})
		case string:
			res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_STRING_VALUE, StringValue: arg.(string)})
		case []byte:
			res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_BYTES_VALUE, BytesValue: arg.([]byte)})
		case [20]byte:
			res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_BYTES_20_VALUE, Bytes20Value: arg.([20]byte)})
		case [32]byte:
			res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_BYTES_32_VALUE, Bytes32Value: arg.([32]byte)})
		case bool:
			res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_BOOL_VALUE, BoolValue: arg.(bool)})
		case *big.Int:
			res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_UINT_256_VALUE, Uint256Value: arg.(*big.Int)})
		}
	}
	return (&protocol.ArgumentArrayBuilder{Arguments: res}).Build()
}

func ArgumentArrayToArgs(ArgumentArray *protocol.ArgumentArray) []interface{} {
	res := []interface{}{}
	for i := ArgumentArray.ArgumentsIterator(); i.HasNext(); {
		Argument := i.NextArguments()
		switch Argument.Type() {
		case protocol.ARGUMENT_TYPE_UINT_32_VALUE:
			res = append(res, Argument.Uint32Value())
		case protocol.ARGUMENT_TYPE_UINT_64_VALUE:
			res = append(res, Argument.Uint64Value())
		case protocol.ARGUMENT_TYPE_STRING_VALUE:
			res = append(res, Argument.StringValue())
		case protocol.ARGUMENT_TYPE_BYTES_VALUE:
			res = append(res, Argument.BytesValue())
		case protocol.ARGUMENT_TYPE_BYTES_20_VALUE:
			res = append(res, Argument.Bytes20Value())
		case protocol.ARGUMENT_TYPE_BYTES_32_VALUE:
			res = append(res, Argument.Bytes32Value())
		case protocol.ARGUMENT_TYPE_BOOL_VALUE:
			res = append(res, Argument.BoolValue())
		case protocol.ARGUMENT_TYPE_UINT_256_VALUE:
			res = append(res, Argument.Uint256Value())
		}
	}
	return res
}
