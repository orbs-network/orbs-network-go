package builders

import (
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"time"
)

// protocol.SignedTransaction

type transaction struct {
	signer  primitives.Ed25519PrivateKey
	builder *protocol.SignedTransactionBuilder
}

func Transaction() *transaction {
	keyPair := keys.Ed25519KeyPairForTests(1)
	return &transaction{
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

func (t *transaction) Build() *protocol.SignedTransaction {
	t.builder.Signature = make([]byte, signature.ED25519_SIGNATURE_SIZE)
	signedTransaction := t.builder.Build()
	sig, err := signature.SignEd25519(t.signer, signedTransaction.Transaction().Raw())
	if err != nil {
		panic(err)
	}
	signedTransaction.MutateSignature(sig)
	return signedTransaction
}

func (t *transaction) Builder() *protocol.SignedTransactionBuilder {
	signedTransaction := t.Build()
	t.builder.Signature = signedTransaction.Signature()
	return t.builder
}

func (t *transaction) WithSigner(publicKey primitives.Ed25519PublicKey, privateKey primitives.Ed25519PrivateKey) *transaction {
	t.builder.Transaction.Signer.Eddsa.SignerPublicKey = publicKey
	t.signer = privateKey
	return t
}

func (t *transaction) WithInvalidSigner(publicKey primitives.Ed25519PublicKey, privateKey primitives.Ed25519PrivateKey) *transaction {
	corruptPrivateKey := make([]byte, len(privateKey))
	return t.WithSigner(publicKey, corruptPrivateKey)
}

func (t *transaction) WithInvalidContent() *transaction {
	t.builder.Transaction.Timestamp = primitives.TimestampNano(time.Now().Add(35 * time.Minute).UnixNano())
	return t
}

func (t *transaction) WithMethod(contractName primitives.ContractName, methodName primitives.MethodName) *transaction {
	t.builder.Transaction.ContractName = contractName
	t.builder.Transaction.MethodName = methodName
	return t
}

func (t *transaction) WithArgs(args ...interface{}) *transaction {
	t.builder.Transaction.InputArguments = MethodArgumentsBuilders(args...)
	return t
}

// BenchmarkToken.transfer

type transferTransaction struct {
	*transaction
}

func TransferTransaction() *transferTransaction {
	return &transferTransaction{Transaction()}
}

func (t *transferTransaction) WithAmount(amount uint64) *transferTransaction {
	t.builder.Transaction.InputArguments[0].Uint64Value = amount
	return t
}
