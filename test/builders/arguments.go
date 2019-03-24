// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package builders

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

/// Test builders for: protocol.ArgumentArray, primitives.PackedArgumentArray

func ArgumentsBuilders(args ...interface{}) (res []*protocol.ArgumentBuilder) {
	res = []*protocol.ArgumentBuilder{}
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
		}
	}
	return
}

func Arguments(args ...interface{}) (res []*protocol.Argument) {
	res = []*protocol.Argument{}
	builders := ArgumentsBuilders(args...)
	for _, builder := range builders {
		res = append(res, builder.Build())
	}
	return
}

func ArgumentsArray(args ...interface{}) *protocol.ArgumentArray {
	res := []*protocol.ArgumentBuilder{}
	builders := ArgumentsBuilders(args...)
	res = append(res, builders...)

	return (&protocol.ArgumentArrayBuilder{Arguments: res}).Build()
}

func PackedArgumentArrayEncode(args ...interface{}) primitives.PackedArgumentArray {
	argArray := ArgumentsArray(args...)
	return argArray.RawArgumentsArray()
}

func PackedArgumentArrayDecode(rawArgumentArrayWithHeader []byte) *protocol.ArgumentArray {
	return protocol.ArgumentArrayReader(rawArgumentArrayWithHeader)
}
