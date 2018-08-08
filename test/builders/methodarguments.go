package builders

import "github.com/orbs-network/orbs-spec/types/go/protocol"

func MethodArguments(args ...interface{}) (res []*protocol.MethodArgument) {
	res = []*protocol.MethodArgument{}
	for _, arg := range args {
		switch arg.(type) {
		case uint32:
			res = append(res, (&protocol.MethodArgumentBuilder{Name: "uint32", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_32_VALUE, Uint32Value: arg.(uint32)}).Build())
		case uint64:
			res = append(res, (&protocol.MethodArgumentBuilder{Name: "uint64", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: arg.(uint64)}).Build())
		case string:
			res = append(res, (&protocol.MethodArgumentBuilder{Name: "string", Type: protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE, StringValue: arg.(string)}).Build())
		case []byte:
			res = append(res, (&protocol.MethodArgumentBuilder{Name: "bytes", Type: protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE, BytesValue: arg.([]byte)}).Build())
		}
	}
	return
}
