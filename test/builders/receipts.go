// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package builders

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

/// Test builders for: protocol.TransactionReceipt

type receipt struct {
	builder *protocol.TransactionReceiptBuilder
}

func TransactionReceipt() *receipt {
	return &receipt{
		builder: &protocol.TransactionReceiptBuilder{
			Txhash:          hash.CalcSha256([]byte("some-tx-hash")),
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

func (r *receipt) WithRandomHash(ctrlRand *rand.ControlledRand) *receipt {
	ctrlRand.Read(r.builder.Txhash)
	return r
}
