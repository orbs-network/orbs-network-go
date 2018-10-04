package gammacli

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

// Bytes are hex and are converted to []byte after json unmarshal
type JSONMethodArgument struct {
	Name  string
	Type  string
	Value interface{}
}

type JSONTransaction struct {
	ContractName string
	MethodName   string
	Arguments    []JSONMethodArgument
}

type TransactionReceipt struct {
	Txhash          primitives.Sha256
	ExecutionResult protocol.ExecutionResult
	OutputArguments []JSONMethodArgument
}

type SendTransactionOutput struct {
	TransactionReceipt TransactionReceipt
	TransactionStatus  protocol.TransactionStatus
	BlockHeight        primitives.BlockHeight
	BlockTimestamp     primitives.TimestampNano
}

type CallMethodOutput struct {
	OutputArguments []JSONMethodArgument
	CallResult      protocol.ExecutionResult
	BlockHeight     primitives.BlockHeight
	BlockTimestamp  primitives.TimestampNano
}

func (ma *JSONMethodArgument) String() string {
	var argumentValue string
	switch ma.Type {
	case METHOD_ARGUMENT_TYPE_UINT32:
		argumentValue = string(ma.Value.(uint32))
	case METHOD_ARGUMENT_TYPE_UINT64:
		argumentValue = string(ma.Value.(uint64))
	case METHOD_ARGUMENT_TYPE_STRING:
		argumentValue = string(ma.Value.(string))
	case METHOD_ARGUMENT_TYPE_BYTES:
		decodedString, _ := hex.DecodeString(ma.Value.(string))
		argumentValue = string(decodedString)
	default:
		argumentValue = "<nil>"
	}

	return ma.Name + ":" + argumentValue
}
