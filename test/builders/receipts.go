package builders

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type receipt struct {
	builder *protocol.TransactionReceiptBuilder
}

func TransactionReceipt() *receipt {
	return &receipt{
		builder: &protocol.TransactionReceiptBuilder{
			Txhash:          []byte("some-tx-hash"),
			ExecutionResult: protocol.EXECUTION_RESULT_SUCCESS,
		},
	}
}

func (r *receipt) Build() *protocol.TransactionReceipt {
	return r.builder.Build()
}

func (r *receipt) Builder() *protocol.TransactionReceiptBuilder {
	return r.builder
}
