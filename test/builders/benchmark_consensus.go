// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package builders

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

/// Test builders for: protocol.BlockPairContainer

func BenchmarkConsensusBlockPair() *blockPair {
	keyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)
	return BlockPair().WithBenchmarkConsensusBlockProof(keyPair)
}

func (b *blockPair) buildBenchmarkConsensusBlockProof(txHeaderBuilt *protocol.TransactionsBlockHeader, rxHeaderBuilt *protocol.ResultsBlockHeader) {
	b.rxProof.BenchmarkConsensus.BlockRef = &consensus.BenchmarkConsensusBlockRefBuilder{
		PlaceholderType: consensus.BENCHMARK_CONSENSUS_VALID,
		BlockHeight:     b.txHeader.BlockHeight,
		PlaceholderView: 1,
		BlockHash: digest.CalcBlockHash(
			&protocol.TransactionsBlockContainer{Header: txHeaderBuilt},
			&protocol.ResultsBlockContainer{Header: rxHeaderBuilt}),
	}
	b.rxProof.TransactionsBlockHash = digest.CalcTransactionsBlockHash(&protocol.TransactionsBlockContainer{Header: txHeaderBuilt})
	sig, err := digest.SignAsNode(b.blockProofSigner, b.rxProof.BenchmarkConsensus.BlockRef.Build().Raw())
	if err != nil {
		panic(err)
	}
	b.rxProof.BenchmarkConsensus.Nodes[0].Signature = sig
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
			BlockRef: nil,
			Nodes: []*consensus.BenchmarkConsensusSenderSignatureBuilder{{
				SenderNodeAddress: keyPair.NodeAddress(),
				Signature:         nil,
			}},
			Placeholder: []byte{0x01, 0x02},
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
	sig, err := digest.SignAsNode(c.messageSigner, statusBuilt.Raw())
	if err != nil {
		panic(err)
	}
	c.sender.Signature = sig
	return &gossipmessages.BenchmarkConsensusCommittedMessage{
		Status: statusBuilt,
		Sender: c.sender.Build(),
	}
}
