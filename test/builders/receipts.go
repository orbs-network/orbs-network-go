package builders

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"math/rand"
)

// protocol.TransactionReceipt

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

func (r *receipt) WithTransaction(t *protocol.Transaction) *receipt {
	r.builder.Txhash = digest.CalcTxHash(t)
	return r
}

func (r *receipt) Build() *protocol.TransactionReceipt {
	return r.builder.Build()
}

func (r *receipt) BuildEmpty() *protocol.TransactionReceipt {
	r.builder.Txhash = []byte{}
	r.builder.ExecutionResult = protocol.EXECUTION_RESULT_RESERVED
	return r.builder.Build()
}

func (r *receipt) Builder() *protocol.TransactionReceiptBuilder {
	return r.builder
}

func (r *receipt) WithRandomHash() *receipt {
	rand.Read(r.builder.Txhash)
	return r
}

func TransactionReceiptOutputArgumentsParse(r *protocol.TransactionReceipt) *protocol.MethodArgumentArrayArgumentsIterator {
	argsArray := protocol.MethodArgumentArrayReader(r.RawOutputArgumentArrayWithHeader())
	return argsArray.ArgumentsIterator()
}
