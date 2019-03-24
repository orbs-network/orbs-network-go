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
	"math"
	"time"
)

/// Test builders for: protocol.SignedTransaction

// do not create this struct directly although it's exported
type TransactionBuilder struct {
	signer  primitives.Ed25519PrivateKey
	builder *protocol.SignedTransactionBuilder
}

func TransferTransaction() *TransactionBuilder {
	keyPair := testKeys.Ed25519KeyPairForTests(1)
	t := &TransactionBuilder{
		signer: keyPair.PrivateKey(),
		builder: &protocol.SignedTransactionBuilder{
			Transaction: &protocol.TransactionBuilder{
				ProtocolVersion: DEFAULT_TEST_PROTOCOL_VERSION,
				VirtualChainId:  DEFAULT_TEST_VIRTUAL_CHAIN_ID,
				ContractName:    "BenchmarkToken",
				MethodName:      "transfer",
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
	return t.WithAmountAndTargetAddress(10, targetAddress)
}

func Transaction() *TransactionBuilder {
	return TransferTransaction()
}

func (t *TransactionBuilder) Build() *protocol.SignedTransaction {
	t.builder.Signature = make([]byte, signature.ED25519_SIGNATURE_SIZE_BYTES)
	signedTransaction := t.builder.Build()
	txHash := digest.CalcTxHash(signedTransaction.Transaction())
	sig, err := signature.SignEd25519(t.signer, txHash)
	if err != nil {
		panic(err)
	}
	signedTransaction.MutateSignature(sig)
	return signedTransaction
}

func (t *TransactionBuilder) Builder() *protocol.SignedTransactionBuilder {
	signedTransaction := t.Build()
	t.builder.Signature = signedTransaction.Signature()
	return t.builder
}

func (t *TransactionBuilder) WithEd25519Signer(keyPair *keys.Ed25519KeyPair) *TransactionBuilder {
	t.builder.Transaction.Signer.Eddsa.SignerPublicKey = keyPair.PublicKey()
	t.signer = keyPair.PrivateKey()
	return t
}

func (t *TransactionBuilder) WithTimestamp(timestamp time.Time) *TransactionBuilder {
	t.builder.Transaction.Timestamp = primitives.TimestampNano(timestamp.UnixNano())
	return t
}

func (t *TransactionBuilder) WithInvalidEd25519Signer(keyPair *keys.Ed25519KeyPair) *TransactionBuilder {
	corruptPrivateKey := make([]byte, len(keyPair.PrivateKey()))
	t.builder.Transaction.Signer.Eddsa.SignerPublicKey = keyPair.PublicKey()
	t.signer = corruptPrivateKey
	return t
}

func (t *TransactionBuilder) WithInvalidPublicKey() *TransactionBuilder {
	keyPair := testKeys.Ed25519KeyPairForTests(1)
	t.builder.Transaction.Signer.Eddsa.SignerPublicKey = keyPair.PublicKey()[1:]
	t.signer = keyPair.PrivateKey()
	return t
}

func (t *TransactionBuilder) WithTimestampInFarFuture() *TransactionBuilder {
	t.builder.Transaction.Timestamp = primitives.TimestampNano(time.Now().Add(300 * time.Minute).UnixNano())
	return t
}

func (t *TransactionBuilder) WithInvalidAmount(targetAddress []byte) *TransactionBuilder {
	return t.WithAmountAndTargetAddress(math.MaxUint64, targetAddress) // Benchmark Contract fails amount over total supply of 1000
}

func (t *TransactionBuilder) WithMethod(contractName primitives.ContractName, methodName primitives.MethodName) *TransactionBuilder {
	t.builder.Transaction.ContractName = contractName
	t.builder.Transaction.MethodName = methodName
	return t
}

func (t *TransactionBuilder) WithArgs(args ...interface{}) *TransactionBuilder {
	t.builder.Transaction.InputArgumentArray = ArgumentsArray(args...).RawArgumentsArray()
	return t
}

func (t *TransactionBuilder) WithAmountAndTargetAddress(amount uint64, targetAddress []byte) *TransactionBuilder {
	return t.WithArgs(amount, targetAddress)
}

func (t *TransactionBuilder) WithTargetAddress(targetAddress []byte) *TransactionBuilder {
	return t.WithArgs(targetAddress)
}

func (t *TransactionBuilder) WithProtocolVersion(v primitives.ProtocolVersion) *TransactionBuilder {
	t.builder.Transaction.ProtocolVersion = v
	return t
}

func (t *TransactionBuilder) WithContract(name string) *TransactionBuilder {
	t.builder.Transaction.ContractName = primitives.ContractName(name)
	return t
}

func (t *TransactionBuilder) WithInvalidSignerScheme() *TransactionBuilder {
	t.builder.Transaction.Signer = &protocol.SignerBuilder{
		Scheme: protocol.SIGNER_SCHEME_EDDSA + 10000,
	}
	return t
}

func (t *TransactionBuilder) WithVirtualChainId(virtualChainId primitives.VirtualChainId) *TransactionBuilder {
	t.builder.Transaction.VirtualChainId = virtualChainId
	return t
}

func TransactionInputArgumentsParse(t *protocol.Transaction) *protocol.ArgumentArrayArgumentsIterator {
	argsArray := protocol.ArgumentArrayReader(t.RawInputArgumentArrayWithHeader())
	return argsArray.ArgumentsIterator()
}

type NonSignedTransactionBuilder struct {
	builder *protocol.TransactionBuilder
}

func NonSignedTransaction() *NonSignedTransactionBuilder {
	keyPair := testKeys.Ed25519KeyPairForTests(1)
	return &NonSignedTransactionBuilder{
		&protocol.TransactionBuilder{
			ProtocolVersion: DEFAULT_TEST_PROTOCOL_VERSION,
			VirtualChainId:  DEFAULT_TEST_VIRTUAL_CHAIN_ID,
			ContractName:    "BenchmarkToken",
			MethodName:      "transfer",
			Signer: &protocol.SignerBuilder{
				Scheme: protocol.SIGNER_SCHEME_EDDSA,
				Eddsa: &protocol.EdDSA01SignerBuilder{
					NetworkType:     protocol.NETWORK_TYPE_TEST_NET,
					SignerPublicKey: keyPair.PublicKey(),
				},
			},
			Timestamp: primitives.TimestampNano(time.Now().UnixNano()),
		},
	}
}

func (t *NonSignedTransactionBuilder) WithMethod(contractName primitives.ContractName, methodName primitives.MethodName) *NonSignedTransactionBuilder {
	t.builder.ContractName = contractName
	t.builder.MethodName = methodName
	return t
}

func (t *NonSignedTransactionBuilder) Build() *protocol.Transaction {
	transaction := t.builder.Build()
	return transaction
}

func (t *NonSignedTransactionBuilder) Builder() *protocol.TransactionBuilder {
	return t.builder
}
