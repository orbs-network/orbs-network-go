package test

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"time"
)

type transferTransaction struct {
	builder *protocol.SignedTransactionBuilder
}

func TransferTransaction() *transferTransaction {
	return &transferTransaction{
		builder: &protocol.SignedTransactionBuilder{
			Transaction: &protocol.TransactionBuilder{
				ContractName: "BenchmarkToken",
				MethodName:   "transfer",
				Timestamp:    primitives.Timestamp(time.Now().Unix()),
				InputArguments: []*protocol.MethodArgumentBuilder{
					{Name: "amount", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: 10},
				},
			},
		},
	}
}

func (t *transferTransaction) Build() *protocol.SignedTransaction {
	return t.builder.Build()
}

func (t *transferTransaction) Builder() *protocol.SignedTransactionBuilder {
	return t.builder
}

func (t *transferTransaction) WithAmount(amount uint64) *transferTransaction {
	t.builder.Transaction.InputArguments[0].Uint64Value = amount
	return t
}

func (t *transferTransaction) WithInvalidContent() *transferTransaction {
	t.builder.Transaction.Timestamp = primitives.Timestamp(time.Now().Add(+35 * time.Minute).Unix())
	return t
}
