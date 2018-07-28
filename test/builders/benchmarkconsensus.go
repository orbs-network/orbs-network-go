package builders

import (
	"github.com/orbs-network/orbs-network-go/crypto"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/logic"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

// protocol.BlockPairContainer

func BenchmarkConsensusBlockPair() *blockPair {
	keyPair := keys.Ed25519KeyPairForTests(0)
	return BlockPair().WithBenchmarkConsensusBlockProof(keyPair.PrivateKey(), keyPair.PublicKey())
}

func (b *blockPair) buildBenchmarkConsensusBlockProof(txHeaderBuilt *protocol.TransactionsBlockHeader, rxHeaderBuilt *protocol.ResultsBlockHeader) {
	txHash := crypto.CalcTransactionsBlockHash(&protocol.BlockPairContainer{TransactionsBlock: &protocol.TransactionsBlockContainer{Header: txHeaderBuilt}})
	rxHash := crypto.CalcResultsBlockHash(&protocol.BlockPairContainer{ResultsBlock: &protocol.ResultsBlockContainer{Header: rxHeaderBuilt}})
	xorHash := logic.CalcXor(txHash, rxHash)
	sig, err := signature.SignEd25519(b.blockProofSigner, xorHash)
	if err != nil {
		panic(err)
	}
	b.rxProof.BenchmarkConsensus.Sender.Signature = sig
}

func (b *blockPair) WithBenchmarkConsensusBlockProof(privateKey primitives.Ed25519PrivateKey, publicKey primitives.Ed25519PublicKey) *blockPair {
	b.blockProofSigner = privateKey
	b.txProof = &protocol.TransactionsBlockProofBuilder{
		Type:               protocol.TRANSACTIONS_BLOCK_PROOF_TYPE_BENCHMARK_CONSENSUS,
		BenchmarkConsensus: &consensus.BenchmarkConsensusBlockProofBuilder{},
	}
	b.rxProof = &protocol.ResultsBlockProofBuilder{
		Type: protocol.RESULTS_BLOCK_PROOF_TYPE_BENCHMARK_CONSENSUS,
		BenchmarkConsensus: &consensus.BenchmarkConsensusBlockProofBuilder{
			Sender: &consensus.BenchmarkConsensusSenderSignatureBuilder{
				SenderPublicKey: publicKey,
				Signature:       nil,
			},
		},
	}
	return b
}

func (b *blockPair) WithInvalidBenchmarkConsensusBlockProof(privateKey primitives.Ed25519PrivateKey, publicKey primitives.Ed25519PublicKey) *blockPair {
	corruptPrivateKey := make([]byte, len(privateKey))
	return b.WithBenchmarkConsensusBlockProof(corruptPrivateKey, publicKey)
}

// gossipmessages.BenchmarkConsensusCommittedMessage

type committed struct {
	messageSigner primitives.Ed25519PrivateKey
	status        *gossipmessages.BenchmarkConsensusStatusBuilder
	sender        *gossipmessages.SenderSignatureBuilder
}

func BenchmarkConsensusCommittedMessage() *committed {
	keyPair := keys.Ed25519KeyPairForTests(0)
	c := &committed{
		status: &gossipmessages.BenchmarkConsensusStatusBuilder{
			LastCommittedBlockHeight: 0,
		},
		sender: &gossipmessages.SenderSignatureBuilder{
			SenderPublicKey: nil,
			Signature:       nil,
		},
	}
	return c.WithSenderSignature(keyPair.PrivateKey(), keyPair.PublicKey())
}

func (c *committed) WithLastCommittedHeight(blockHeight primitives.BlockHeight) *committed {
	c.status.LastCommittedBlockHeight = blockHeight
	return c
}

func (c *committed) WithSenderSignature(privateKey primitives.Ed25519PrivateKey, publicKey primitives.Ed25519PublicKey) *committed {
	c.messageSigner = privateKey
	c.sender.SenderPublicKey = publicKey
	return c
}

func (c *committed) WithInvalidSenderSignature(privateKey primitives.Ed25519PrivateKey, publicKey primitives.Ed25519PublicKey) *committed {
	return c.WithSenderSignature(make([]byte, len(privateKey)), publicKey)
}

func (c *committed) Build() *gossipmessages.BenchmarkConsensusCommittedMessage {
	statusBuilt := c.status.Build()
	signedData := hash.CalcSha256(statusBuilt.Raw())
	sig, err := signature.SignEd25519(c.messageSigner, signedData)
	if err != nil {
		panic(err)
	}
	c.sender.Signature = sig
	return &gossipmessages.BenchmarkConsensusCommittedMessage{
		Status: statusBuilt,
		Sender: c.sender.Build(),
	}
}
