package jsonapi

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

// Bytes are hex and are converted to []byte after json unmarshal
type MethodArgument struct {
	Name  string
	Type  string
	Value interface{}
}

type Transaction struct {
	ContractName string
	MethodName   string
	Arguments    []MethodArgument
}

type TransactionReceipt struct {
	Txhash          primitives.Sha256
	ExecutionResult protocol.ExecutionResult
	OutputArguments []MethodArgument
}

type SendTransactionOutput struct {
	TransactionReceipt TransactionReceipt
	TransactionStatus  protocol.TransactionStatus
	BlockHeight        primitives.BlockHeight
	BlockTimestamp     primitives.TimestampNano
}

type CallMethodOutput struct {
	OutputArguments []MethodArgument
	CallResult      protocol.ExecutionResult
	BlockHeight     primitives.BlockHeight
	BlockTimestamp  primitives.TimestampNano
}

func (ma *MethodArgument) String() string {
	returnString := ma.Name + ":"

	switch ma.Type {
	case METHOD_ARGUMENT_TYPE_UINT32:
		returnString = returnString + string(ma.Value.(uint32))
	case METHOD_ARGUMENT_TYPE_UINT64:
		returnString = returnString + string(ma.Value.(uint64))
	case METHOD_ARGUMENT_TYPE_STRING:
		returnString = returnString + string(ma.Value.(string))
	case METHOD_ARGUMENT_TYPE_BYTES:
		decodedString, _ := hex.DecodeString(ma.Value.(string))
		returnString = returnString + string(decodedString)
	default:
		returnString = returnString + "<nil>"
	}

	return returnString
}
