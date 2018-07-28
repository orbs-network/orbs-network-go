package test

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
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
		keyPair := keys.Ed25519KeyPairForTests(i + 1) // leader is set 0
		var c *gossipmessages.BenchmarkConsensusCommittedMessage
		if validSignature {
			c = aCommitted.WithSenderSignature(keyPair.PrivateKey(), keyPair.PublicKey()).Build()
		} else {
			c = aCommitted.WithInvalidSenderSignature(keyPair.PrivateKey(), keyPair.PublicKey()).Build()
		}
		h.receivedCommittedViaGossip(c)
	}
}
