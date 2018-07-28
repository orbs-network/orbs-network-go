package test

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

func (h *harness) receivedCommittedViaGossip(message *gossipmessages.BenchmarkConsensusCommittedMessage) {
	h.service.HandleBenchmarkConsensusCommitted(&gossiptopics.BenchmarkConsensusCommittedInput{
		RecipientPublicKey: nil,
		Message:            message,
	})
}

func (h *harness) receivedCommittedViaGossipFromSeveral(numNodes int, lastCommitted primitives.BlockHeight, validSignature bool) {
	aCommitted := builders.BenchmarkConsensusCommittedMessage().WithLastCommittedHeight(lastCommitted)
	for i := 0; i < numNodes; i++ {
		keyPair := crypto.Ed25519KeyPairForTests(i + 2) // leader is set 1
		var c *gossipmessages.BenchmarkConsensusCommittedMessage
		if validSignature {
			c = aCommitted.WithSenderSignature(keyPair.PrivateKeyUnsafe(), keyPair.PublicKey()).Build()
		} else {
			c = aCommitted.WithInvalidSenderSignature(keyPair.PrivateKeyUnsafe(), keyPair.PublicKey()).Build()
		}
		h.receivedCommittedViaGossip(c)
	}
}
