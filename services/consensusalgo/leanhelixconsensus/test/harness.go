// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/signer"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/leanhelixconsensus"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/stretchr/testify/require"
	"time"
)

const NETWORK_SIZE = 4

type harness struct {
	consensus              *leanhelixconsensus.Service
	gossip                 *gossiptopics.MockLeanHelix
	blockStorage           *services.MockBlockStorage
	consensusContext       *services.MockConsensusContext
	instanceId             lhprimitives.InstanceId
	auditBlocksYoungerThan time.Duration
}

func newLeanHelixServiceHarness(auditBlocksYoungerThan time.Duration) *harness {
	h := &harness{
		gossip:                 &gossiptopics.MockLeanHelix{},
		blockStorage:           &services.MockBlockStorage{},
		consensusContext:       &services.MockConsensusContext{},
		auditBlocksYoungerThan: auditBlocksYoungerThan,
	}

	h.resetAndApplyMockDefaults()

	return h
}

func (h *harness) resetAndApplyMockDefaults() {
	h.consensusContext.Reset()
	h.blockStorage.Reset()
	h.gossip.Reset()

	h.blockStorage.When("RegisterConsensusBlocksHandler", mock.Any).Return().Times(1)
	h.gossip.When("RegisterLeanHelixHandler", mock.Any).Return().Times(1)
}

func (h *harness) start(parent *with.ConcurrencyHarness, ctx context.Context) *harness {
	registry := metric.NewRegistry()

	cfg := config.ForLeanHelixConsensusTests(testKeys.EcdsaSecp256K1KeyPairForTests(0), h.auditBlocksYoungerThan)
	h.instanceId = leanhelixconsensus.CalcInstanceId(cfg.NetworkType(), cfg.VirtualChainId())

	signer, err := signer.New(cfg)
	require.NoError(parent.T, err)

	h.consensus = leanhelixconsensus.NewLeanHelixConsensusAlgo(ctx, h.gossip, h.blockStorage, h.consensusContext, signer, parent.Logger, cfg, registry)
	parent.Supervise(h.consensus)
	return h
}

func (h *harness) getCommitteeWithNodeIndexAsLeader(nodeIndex int) []primitives.NodeAddress {
	res := []primitives.NodeAddress{
		testKeys.EcdsaSecp256K1KeyPairForTests(nodeIndex).NodeAddress(),
	}
	for i := 0; i < NETWORK_SIZE; i++ {
		if i != nodeIndex {
			res = append(res, testKeys.EcdsaSecp256K1KeyPairForTests(i).NodeAddress())
		}
	}
	return res
}

func (h *harness) beLastInCommittee() {
	h.expectConsensusContextRequestOrderingCommittee(1)
}

func (h *harness) beFirstInCommittee() {
	h.expectConsensusContextRequestOrderingCommittee(0)
}

func (h *harness) expectConsensusContextRequestOrderingCommittee(leaderNodeIndex int) {
	h.consensusContext.When("RequestOrderingCommittee", mock.Any, mock.Any).Return(&services.RequestCommitteeOutput{
		NodeAddresses: h.getCommitteeWithNodeIndexAsLeader(leaderNodeIndex),
	}, nil).Times(1)
}

func (h *harness) expectConsensusContextRequestBlock(blockPair *protocol.BlockPairContainer) {
	h.consensusContext.When("RequestNewTransactionsBlock", mock.Any, mock.Any).Return(&services.RequestNewTransactionsBlockOutput{
		TransactionsBlock: blockPair.TransactionsBlock,
	}, nil).Times(1)
	h.consensusContext.When("RequestNewResultsBlock", mock.Any, mock.Any).Return(&services.RequestNewResultsBlockOutput{
		ResultsBlock: blockPair.ResultsBlock,
	}, nil).Times(1)
}

func (h *harness) expectGossipSendLeanHelixMessage() {
	h.gossip.When("SendLeanHelixMessage", mock.Any, mock.Any).Return(nil, nil) // TODO Maybe add .Times(1) like there was before
}

func (h *harness) expectNeverToProposeABlock() {
	h.consensusContext.Never("RequestNewTransactionsBlock", mock.Any, mock.Any)
	h.consensusContext.Never("RequestNewResultsBlock", mock.Any, mock.Any)
}
