// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package builders

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"time"
)

/// Test builders for: client.TransactionRef

type transactionRef struct {
	builder *client.TransactionRefBuilder
}

func TransactionRef() *transactionRef {
	return &transactionRef{
		builder: &client.TransactionRefBuilder{
			ProtocolVersion:      DEFAULT_TEST_PROTOCOL_VERSION,
			VirtualChainId:       DEFAULT_TEST_VIRTUAL_CHAIN_ID,
			TransactionTimestamp: primitives.TimestampNano(time.Now().UnixNano()),
			Txhash:               hash.CalcSha256([]byte("some-tx-hash")),
		},
	}
}

func (r *transactionRef) Build() *client.TransactionRef {
	return r.builder.Build()
}

func (r *transactionRef) Builder() *client.TransactionRefBuilder {
	return r.builder
}

func (r *transactionRef) WithTxHash(txHash primitives.Sha256) *transactionRef {
	r.builder.Txhash = txHash
	return r
}
