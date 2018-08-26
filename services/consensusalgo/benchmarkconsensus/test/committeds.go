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

func (h *harness) receivedCommittedMessagesViaGossip(msgs []*gossipmessages.BenchmarkConsensusCommittedMessage) {
	for _, msg := range msgs {
		h.receivedCommittedViaGossip(msg)
	}
}

// builder

type committed struct {
	count                int
	blockHeight          primitives.BlockHeight
	invalidSignatures    bool
	nonFederationMembers bool
}

func committedMessages() *committed {
	return &committed{}
}

func (c *committed) WithCountBelowQuorum() *committed {
	c.count = 2
	return c
}

func (c *committed) WithCountAboveQuorum() *committed {
	//TODO: Change count to 3 once requiredQuorumSize is 2/3 
	c.count = 4
	return c
}

func (c *committed) WithHeight(blockHeight primitives.BlockHeight) *committed {
	c.blockHeight = blockHeight
	return c
}

func (c *committed) WithInvalidSignatures() *committed {
	c.invalidSignatures = true
	return c
}

func (c *committed) FromNonFederationMembers() *committed {
	c.nonFederationMembers = true
	return c
}

func (c *committed) Build() (res []*gossipmessages.BenchmarkConsensusCommittedMessage) {
	aCommitted := builders.BenchmarkConsensusCommittedMessage().WithLastCommittedHeight(c.blockHeight)
	for i := 0; i < c.count; i++ {
		keyPair := keys.Ed25519KeyPairForTests(i + 1) // leader is set 0
		if c.nonFederationMembers {
			keyPair = keys.Ed25519KeyPairForTests(i + networkSize)
		}
		if c.invalidSignatures {
			res = append(res, aCommitted.WithInvalidSenderSignature(keyPair).Build())
		} else {
			res = append(res, aCommitted.WithSenderSignature(keyPair).Build())
		}
	}
	return
}
