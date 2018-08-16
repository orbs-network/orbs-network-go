package jsonapi

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

//TODO []byte are marshalled as base64. Should we use base58?

type MethodArgument struct {
	Name string
	Type protocol.MethodArgumentType
	Uint32Value uint32
	Uint64Value uint64
	StringValue string
	BytesValue []byte
}

type Transaction struct {
	ContractName string
	MethodName string
	Arguments []MethodArgument
}

type TransactionReceipt struct {
	Txhash primitives.Sha256
	ExecutionResult protocol.ExecutionResult
	OutputArguments []MethodArgument
}

type SendTransactionOutput struct {
	TransactionReceipt TransactionReceipt
	TransactionStatus protocol.TransactionStatus
	BlockHeight primitives.BlockHeight
	BlockTimestamp primitives.TimestampNano
}

type CallMethodOutput struct {
	OutputArguments []MethodArgument
	CallResult protocol.ExecutionResult
	BlockHeight primitives.BlockHeight
	BlockTimestamp primitives.TimestampNano
}
