package builders

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	"github.com/orbs-network/orbs-network-go/crypto/logic"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

// protocol.BlockPairContainer

func BenchmarkConsensusBlockPair() *blockPair {
	keyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)
	return BlockPair().WithBenchmarkConsensusBlockProof(keyPair)
}

func (b *blockPair) buildBenchmarkConsensusBlockProof(txHeaderBuilt *protocol.TransactionsBlockHeader, rxHeaderBuilt *protocol.ResultsBlockHeader) {
	txHash := digest.CalcTransactionsBlockHash(&protocol.TransactionsBlockContainer{Header: txHeaderBuilt})
	rxHash := digest.CalcResultsBlockHash(&protocol.ResultsBlockContainer{Header: rxHeaderBuilt})
	xorHash := logic.CalcXor(txHash, rxHash)
	sig, err := signature.SignEcdsaSecp256K1(b.blockProofSigner, xorHash)
	if err != nil {
		panic(err)
	}
	b.rxProof.BenchmarkConsensus.Sender.Signature = sig
}

func (b *blockPair) WithBenchmarkConsensusBlockProof(keyPair *testKeys.TestEcdsaSecp256K1KeyPair) *blockPair {
	b.blockProofSigner = keyPair.PrivateKey()
	b.txProof = &protocol.TransactionsBlockProofBuilder{
		Type:               protocol.TRANSACTIONS_BLOCK_PROOF_TYPE_BENCHMARK_CONSENSUS,
		BenchmarkConsensus: &consensus.BenchmarkConsensusBlockProofBuilder{},
	}
	b.rxProof = &protocol.ResultsBlockProofBuilder{
		Type: protocol.RESULTS_BLOCK_PROOF_TYPE_BENCHMARK_CONSENSUS,
		BenchmarkConsensus: &consensus.BenchmarkConsensusBlockProofBuilder{
			Sender: &consensus.BenchmarkConsensusSenderSignatureBuilder{
				SenderNodeAddress: keyPair.NodeAddress(),
				Signature:         nil,
			},
		},
	}
	return b
}

func (b *blockPair) WithInvalidBenchmarkConsensusBlockProof(keyPair *testKeys.TestEcdsaSecp256K1KeyPair) *blockPair {
	corruptPrivateKey := make([]byte, len(keyPair.PrivateKey()))
	copy(corruptPrivateKey, keyPair.PrivateKey())
	corruptPrivateKey[5] ^= 0x55
	corruptKeyPair := keys.NewEcdsaSecp256K1KeyPair(keyPair.PublicKey(), corruptPrivateKey)
	return b.WithBenchmarkConsensusBlockProof(&testKeys.TestEcdsaSecp256K1KeyPair{corruptKeyPair})
}

// gossipmessages.BenchmarkConsensusCommittedMessage

type committed struct {
	messageSigner primitives.EcdsaSecp256K1PrivateKey
	status        *gossipmessages.BenchmarkConsensusStatusBuilder
	sender        *gossipmessages.SenderSignatureBuilder
}

func BenchmarkConsensusCommittedMessage() *committed {
	keyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)
	c := &committed{
		status: &gossipmessages.BenchmarkConsensusStatusBuilder{
			LastCommittedBlockHeight: 0,
		},
		sender: &gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: nil,
			Signature:         nil,
		},
	}
	return c.WithSenderSignature(keyPair)
}

func (c *committed) WithLastCommittedHeight(blockHeight primitives.BlockHeight) *committed {
	c.status.LastCommittedBlockHeight = blockHeight
	return c
}

func (c *committed) WithSenderSignature(keyPair *testKeys.TestEcdsaSecp256K1KeyPair) *committed {
	c.messageSigner = keyPair.PrivateKey()
	c.sender.SenderNodeAddress = keyPair.NodeAddress()
	return c
}

func (c *committed) WithInvalidSenderSignature(keyPair *testKeys.TestEcdsaSecp256K1KeyPair) *committed {
	corruptPrivateKey := make([]byte, len(keyPair.PrivateKey()))
	copy(corruptPrivateKey, keyPair.PrivateKey())
	corruptPrivateKey[5] ^= 0x55
	corruptKeyPair := keys.NewEcdsaSecp256K1KeyPair(keyPair.PublicKey(), corruptPrivateKey)
	return c.WithSenderSignature(&testKeys.TestEcdsaSecp256K1KeyPair{corruptKeyPair})
}

func (c *committed) Build() *gossipmessages.BenchmarkConsensusCommittedMessage {
	statusBuilt := c.status.Build()
	signedData := hash.CalcSha256(statusBuilt.Raw())
	sig, err := signature.SignEcdsaSecp256K1(c.messageSigner, signedData)
	if err != nil {
		panic(err)
	}
	c.sender.Signature = sig
	return &gossipmessages.BenchmarkConsensusCommittedMessage{
		Status: statusBuilt,
		Sender: c.sender.Build(),
	}
}
