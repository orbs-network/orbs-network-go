package builders

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"time"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
)

// protocol.SignedTransaction

type TransferTransactionBuilder struct {
	builder *protocol.SignedTransactionBuilder
}

func TransferTransaction() *TransferTransactionBuilder {
	return (&TransferTransactionBuilder{
		builder: &protocol.SignedTransactionBuilder{
			Transaction: &protocol.TransactionBuilder{
				MethodName: "transfer",
				Timestamp:  primitives.TimestampNano(time.Now().UnixNano()),
				InputArguments: []*protocol.MethodArgumentBuilder{
					{Name: "amount", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: 10},
				},
			},
		},
	}).
		WithSigner(protocol.NETWORK_TYPE_TEST_NET, primitives.Ed25519PublicKey(keys.Ed25519KeyPairForTests(1).PublicKey())).
		WithContract("BenchmarkToken").
		WithProtocolVersion(1).
		WithVirtualChainId(primitives.VirtualChainId(42))
}

func (t *TransferTransactionBuilder) Build() *protocol.SignedTransaction {
	return t.builder.Build()
}

func (t *TransferTransactionBuilder) Builder() *protocol.SignedTransactionBuilder {
	return t.builder
}

func (t *TransferTransactionBuilder) WithAmount(amount uint64) *TransferTransactionBuilder {
	t.builder.Transaction.InputArguments[0].Uint64Value = amount
	return t
}

func (t *TransferTransactionBuilder) WithInvalidContent() *TransferTransactionBuilder {
	t.builder.Transaction.Timestamp = primitives.TimestampNano(time.Now().Add(-100000 * time.Hour).UnixNano())
	return t
}

func (t *TransferTransactionBuilder) WithProtocolVersion(v primitives.ProtocolVersion) *TransferTransactionBuilder {
	t.builder.Transaction.ProtocolVersion = v
	return t
}
func (t *TransferTransactionBuilder) WithSigner(networkType protocol.SignerNetworkType, publicKey primitives.Ed25519PublicKey) *TransferTransactionBuilder {
	t.builder.Transaction.Signer = &protocol.SignerBuilder{
		Scheme: protocol.SIGNER_SCHEME_EDDSA,
		Eddsa: &protocol.EdDSA01SignerBuilder{
			NetworkType:     networkType,
			SignerPublicKey: publicKey,
		},
	}
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

func (t *TransferTransactionBuilder) WithTimestamp(timestamp time.Time) *TransferTransactionBuilder {
	t.builder.Transaction.Timestamp = primitives.TimestampNano(timestamp.UnixNano())
	return t
}

func (t *TransferTransactionBuilder) WithVirtualChainId(virtualChainId primitives.VirtualChainId) *TransferTransactionBuilder {
	t.builder.Transaction.VirtualChainId = virtualChainId
	return t
}
