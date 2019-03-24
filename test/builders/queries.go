// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package builders

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"time"
)

/// Test builders for: protocol.SignedQuery

// do not create this struct directly although it's exported
type QueryBuilder struct {
	signer  primitives.Ed25519PrivateKey
	builder *protocol.SignedQueryBuilder
}

func GetBalanceQuery() *QueryBuilder {
	keyPair := testKeys.Ed25519KeyPairForTests(1)
	q := &QueryBuilder{
		signer: keyPair.PrivateKey(),
		builder: &protocol.SignedQueryBuilder{
			Query: &protocol.QueryBuilder{
				ProtocolVersion: DEFAULT_TEST_PROTOCOL_VERSION,
				VirtualChainId:  DEFAULT_TEST_VIRTUAL_CHAIN_ID,
				ContractName:    "BenchmarkToken",
				MethodName:      "getBalance",
				Signer: &protocol.SignerBuilder{
					Scheme: protocol.SIGNER_SCHEME_EDDSA,
					Eddsa: &protocol.EdDSA01SignerBuilder{
						NetworkType:     protocol.NETWORK_TYPE_TEST_NET,
						SignerPublicKey: keyPair.PublicKey(),
					},
				},
				Timestamp: primitives.TimestampNano(time.Now().UnixNano()),
			},
		},
	}
	targetAddress := ClientAddressForEd25519SignerForTests(2)
	return q.WithTargetAddress(targetAddress)
}

func Query() *QueryBuilder {
	return GetBalanceQuery()
}

func (q *QueryBuilder) Build() *protocol.SignedQuery {
	q.builder.Signature = make([]byte, signature.ED25519_SIGNATURE_SIZE_BYTES)
	signedQuery := q.builder.Build()
	queryHash := digest.CalcQueryHash(signedQuery.Query())
	sig, err := signature.SignEd25519(q.signer, queryHash)
	if err != nil {
		panic(err)
	}
	signedQuery.MutateSignature(sig)
	return signedQuery
}

func (q *QueryBuilder) Builder() *protocol.SignedQueryBuilder {
	signedQuery := q.Build()
	q.builder.Signature = signedQuery.Signature()
	return q.builder
}

func (q *QueryBuilder) WithEd25519Signer(keyPair *keys.Ed25519KeyPair) *QueryBuilder {
	q.builder.Query.Signer.Eddsa.SignerPublicKey = keyPair.PublicKey()
	q.signer = keyPair.PrivateKey()
	return q
}

func (q *QueryBuilder) WithMethod(contractName primitives.ContractName, methodName primitives.MethodName) *QueryBuilder {
	q.builder.Query.ContractName = contractName
	q.builder.Query.MethodName = methodName
	return q
}

func (q *QueryBuilder) WithArgs(args ...interface{}) *QueryBuilder {
	q.builder.Query.InputArgumentArray = ArgumentsArray(args...).RawArgumentsArray()
	return q
}

func (q *QueryBuilder) WithAmountAndTargetAddress(amount uint64, targetAddress []byte) *QueryBuilder {
	return q.WithArgs(amount, targetAddress)
}

func (q *QueryBuilder) WithTargetAddress(targetAddress []byte) *QueryBuilder {
	return q.WithArgs(targetAddress)
}

func (q *QueryBuilder) WithVirtualChainId(virtualChainId primitives.VirtualChainId) *QueryBuilder {
	q.builder.Query.VirtualChainId = virtualChainId
	return q
}
