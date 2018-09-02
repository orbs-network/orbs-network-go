package builders

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"time"
)

const DEFAULT_TEST_VIRTUAL_CHAIN_ID = primitives.VirtualChainId(42)

type TransactionBuilder struct {
	signer  primitives.Ed25519PrivateKey
	builder *protocol.SignedTransactionBuilder
}

func TransferTransaction() *TransactionBuilder {
	keyPair := keys.Ed25519KeyPairForTests(1)
	t := &TransactionBuilder{
		signer: keyPair.PrivateKey(),
		builder: &protocol.SignedTransactionBuilder{
			Transaction: &protocol.TransactionBuilder{
				ProtocolVersion: 1,
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
	return t.WithAmount(10)
}

func Transaction() *TransactionBuilder {
	return TransferTransaction()
}

func (t *TransactionBuilder) Build() *protocol.SignedTransaction {
	t.builder.Signature = make([]byte, signature.ED25519_SIGNATURE_SIZE)
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
	keyPair := keys.Ed25519KeyPairForTests(1)
	t.builder.Transaction.Signer.Eddsa.SignerPublicKey = keyPair.PublicKey()[1:]
	t.signer = keyPair.PrivateKey()
	return t
}

func (t *TransactionBuilder) WithInvalidContent() *TransactionBuilder {
	t.builder.Transaction.Timestamp = primitives.TimestampNano(time.Now().Add(35 * time.Minute).UnixNano())
	return t
}

func (t *TransactionBuilder) WithMethod(contractName primitives.ContractName, methodName primitives.MethodName) *TransactionBuilder {
	t.builder.Transaction.ContractName = contractName
	t.builder.Transaction.MethodName = methodName
	return t
}

func (t *TransactionBuilder) WithArgs(args ...interface{}) *TransactionBuilder {
	t.builder.Transaction.InputArgumentArray = MethodArgumentsArray(args...).RawArgumentsArray()
	return t
}

func (t *TransactionBuilder) WithAmount(amount uint64) *TransactionBuilder {
	return t.WithArgs(amount)
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

func TransactionInputArgumentsParse(t *protocol.Transaction) *protocol.MethodArgumentArrayArgumentsIterator {
	argsArray := protocol.MethodArgumentArrayReader(t.RawInputArgumentArrayWithHeader())
	return argsArray.ArgumentsIterator()
}
