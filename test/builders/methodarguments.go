package builders

import "github.com/orbs-network/orbs-spec/types/go/protocol"

func MethodArgumentsBuilders(args ...interface{}) (res []*protocol.MethodArgumentBuilder) {
	res = []*protocol.MethodArgumentBuilder{}
	for _, arg := range args {
		switch arg.(type) {
		case uint32:
			res = append(res, &protocol.MethodArgumentBuilder{Name: "uint32", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_32_VALUE, Uint32Value: arg.(uint32)})
		case uint64:
			res = append(res, &protocol.MethodArgumentBuilder{Name: "uint64", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: arg.(uint64)})
		case string:
			res = append(res, &protocol.MethodArgumentBuilder{Name: "string", Type: protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE, StringValue: arg.(string)})
		case []byte:
			res = append(res, &protocol.MethodArgumentBuilder{Name: "bytes", Type: protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE, BytesValue: arg.([]byte)})
		}
	}
	return
}

func MethodArguments(args ...interface{}) (res []*protocol.MethodArgument) {
	res = []*protocol.MethodArgument{}
	builders := MethodArgumentsBuilders(args...)
	for _, builder := range builders {
		res = append(res, builder.Build())
	}
	return
}
