// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/benchmarkconsensus"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

func (h *harness) receivedCommittedViaGossip(ctx context.Context, message *gossipmessages.BenchmarkConsensusCommittedMessage) {
	h.service.HandleBenchmarkConsensusCommitted(ctx, &gossiptopics.BenchmarkConsensusCommittedInput{
		RecipientNodeAddress: nil,
		Message:              message,
	})
}

func (h *harness) receivedCommittedMessagesViaGossip(ctx context.Context, msgs []*gossipmessages.BenchmarkConsensusCommittedMessage) {
	for _, msg := range msgs {
		h.receivedCommittedViaGossip(ctx, msg)
	}
}

// builder

type committed struct {
	count                int
	blockHeight          primitives.BlockHeight
	invalidSignatures    bool
	nonGenesisValidators bool
}

func multipleCommittedMessages() *committed {
	return &committed{}
}

func (c *committed) WithCountBelowQuorum(cfg benchmarkconsensus.Config) *committed {
	networkSize := len(cfg.GenesisValidatorNodes())
	if networkSize != 5 || cfg.BenchmarkConsensusRequiredQuorumPercentage() != 66 {
		panic("tests assumes 5 nodes with quorum percentage of 66, quorum is 4/5 = 3 other nodes")
	}
	c.count = 2
	return c
}

func (c *committed) WithCountAboveQuorum(cfg benchmarkconsensus.Config) *committed {
	networkSize := len(cfg.GenesisValidatorNodes())
	if networkSize != 5 || cfg.BenchmarkConsensusRequiredQuorumPercentage() != 66 {
		panic("tests assumes 5 nodes with quorum percentage of 66, quorum is 4/5 = 3 other nodes")
	}
	c.count = 3
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

func (c *committed) FromNonGenesisValidators() *committed {
	c.nonGenesisValidators = true
	return c
}

func (c *committed) Build() (res []*gossipmessages.BenchmarkConsensusCommittedMessage) {
	aCommitted := builders.BenchmarkConsensusCommittedMessage().WithLastCommittedHeight(c.blockHeight)
	for i := 0; i < c.count; i++ {
		keyPair := keys.EcdsaSecp256K1KeyPairForTests(i + 1) // leader is set 0
		if c.nonGenesisValidators {
			keyPair = keys.EcdsaSecp256K1KeyPairForTests(i + NETWORK_SIZE)
		}
		if c.invalidSignatures {
			res = append(res, aCommitted.WithInvalidSenderSignature(keyPair).Build())
		} else {
			res = append(res, aCommitted.WithSenderSignature(keyPair).Build())
		}
	}
	return
}
