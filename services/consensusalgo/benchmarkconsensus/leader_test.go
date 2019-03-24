// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package benchmarkconsensus

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLeaderQuorum(t *testing.T) {
	nodes := make(map[string]config.ValidatorNode)

	for i := 0; i < 6; i++ {
		nodes[fmt.Sprintf("fake-key-node%d", i)] = nil
	}

	cfg := config.ForProduction("")
	cfg.SetGenesisValidatorNodes(nodes)

	require.NotZero(t, len(cfg.GenesisValidatorNodes()))

	s := &service{
		config: cfg,
	}

	require.NotZero(t, s.requiredQuorumSize())
}

type fakeFed struct{}

func (f *fakeFed) NodeAddress() primitives.NodeAddress {
	return []byte("bbbabababab")
}

func TestLeaderBadKey(t *testing.T) {
	nodes := make(map[string]config.ValidatorNode)

	for i := 1; i < 6; i++ {
		nodes[fmt.Sprintf("fake-key-node%d", i)] = nil
	}
	fake := &fakeFed{}
	nodes["fake-key-node0"] = fake

	cfg := config.ForProduction("")
	cfg.SetGenesisValidatorNodes(nodes)

	s := &service{
		config: cfg,
	}

	require.Panics(t, func() {
		s.leaderGenerateGenesisBlock()
	}, "should panic")
}
