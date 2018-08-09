package builders

import (
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"time"
)

// protocol.SignedTransaction

type TransferTransactionBuilder struct {
	signer  primitives.Ed25519PrivateKey
	builder *protocol.SignedTransactionBuilder
}

func TransferTransaction() *TransferTransactionBuilder {
	keyPair := keys.Ed25519KeyPairForTests(1)
	return &TransferTransactionBuilder{
		signer: keyPair.PrivateKey(),
		builder: &protocol.SignedTransactionBuilder{
			Transaction: &protocol.TransactionBuilder{
				ProtocolVersion: 1,
				VirtualChainId:  primitives.VirtualChainId(42),
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
				InputArguments: []*protocol.MethodArgumentBuilder{
					{Name: "amount", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: 10},
				},
			},
		},
	}
}

func (t *TransferTransactionBuilder) Build() *protocol.SignedTransaction {
	t.builder.Signature = make([]byte, signature.ED25519_SIGNATURE_SIZE)
	signedTransaction := t.builder.Build()
	sig, err := signature.SignEd25519(t.signer, signedTransaction.Transaction().Raw())
	if err != nil {
		panic(err)
	}
	signedTransaction.MutateSignature(sig)
	return signedTransaction
}

func (t *TransferTransactionBuilder) Builder() *protocol.SignedTransactionBuilder {
	signedTransaction := t.Build()
	t.builder.Signature = signedTransaction.Signature()
	return t.builder
}

func (t *TransferTransactionBuilder) WithSigner(keyPair keys.Ed25519KeyPair) *TransferTransactionBuilder {
	t.builder.Transaction.Signer.Eddsa.SignerPublicKey = keyPair.PublicKey()
	t.signer = keyPair.PrivateKey()
	return t
}

func (t *TransferTransactionBuilder) WithTimestamp(timestamp time.Time) *TransferTransactionBuilder {
	t.builder.Transaction.Timestamp = primitives.TimestampNano(timestamp.UnixNano())
	return t
}

func (t *TransferTransactionBuilder) WithInvalidSigner() *TransferTransactionBuilder {
	keyPair := keys.Ed25519KeyPairForTests(1)
	corruptPrivateKey := make([]byte, len(keyPair.PrivateKey()))
	t.builder.Transaction.Signer.Eddsa.SignerPublicKey = keyPair.PublicKey()
	t.signer = corruptPrivateKey
	return t
}

func (t *TransferTransactionBuilder) WithInvalidPublicKey() *TransferTransactionBuilder {
	keyPair := keys.Ed25519KeyPairForTests(1)
	t.builder.Transaction.Signer.Eddsa.SignerPublicKey = keyPair.PublicKey()[1:]
	t.signer = keyPair.PrivateKey()
	return t
}

func (t *TransferTransactionBuilder) WithInvalidContent() *TransferTransactionBuilder {
	t.builder.Transaction.Timestamp = primitives.TimestampNano(time.Now().Add(35 * time.Minute).UnixNano())
	return t
}

func (t *TransferTransactionBuilder) WithMethod(contractName primitives.ContractName, methodName primitives.MethodName) *TransferTransactionBuilder {
	t.builder.Transaction.ContractName = contractName
	t.builder.Transaction.MethodName = methodName
	return t
}

func (t *TransferTransactionBuilder) WithArgs(args ...interface{}) *TransferTransactionBuilder {
	t.builder.Transaction.InputArguments = MethodArgumentsBuilders(args...)
	return t
}

func (t *TransferTransactionBuilder) WithAmount(amount uint64) *TransferTransactionBuilder {
	t.builder.Transaction.InputArguments[0].Uint64Value = amount
	return t
}

func (t *TransferTransactionBuilder) WithProtocolVersion(v primitives.ProtocolVersion) *TransferTransactionBuilder {
	t.builder.Transaction.ProtocolVersion = v
	return t
}

func (t *TransferTransactionBuilder) WithContract(name string) *TransferTransactionBuilder {
	t.builder.Transaction.ContractName = primitives.ContractName(name)
	return t
}

func (t *TransferTransactionBuilder) WithInvalidSignerScheme() *TransferTransactionBuilder {
	t.builder.Transaction.Signer = &protocol.SignerBuilder{
		Scheme: protocol.SIGNER_SCHEME_EDDSA + 10000,
	}
	return t
}

func (t *TransferTransactionBuilder) WithVirtualChainId(virtualChainId primitives.VirtualChainId) *TransferTransactionBuilder {
	t.builder.Transaction.VirtualChainId = virtualChainId
	return t
}
