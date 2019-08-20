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
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const NETWORK_SIZE = 4

type harness struct {
	consensus              services.ConsensusAlgoLeanHelix
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

	h.resetMocks()

	return h
}

func (h *harness) resetMocks() {
	h.ResetConsensusContextMock()
	h.ResetBlockStorageMock()
	h.ResetGossipMock()
}

func (h *harness) ResetConsensusContextMock() {
	h.consensusContext.Reset()
}

func (h *harness) ResetBlockStorageMock() {
	h.blockStorage.Reset()
	h.blockStorage.When("RegisterConsensusBlocksHandler", mock.Any).Return().Times(1)
}

func (h *harness) ResetGossipMock() {
	h.gossip.Reset()
	h.gossip.When("RegisterLeanHelixHandler", mock.Any).Return().Times(1)
}

func (h *harness) start(tb testing.TB, ctx context.Context) *harness {
	logOutput := log.NewTestOutput(tb, log.NewHumanReadableFormatter())
	logger := log.GetLogger().WithOutput(logOutput)
	registry := metric.NewRegistry()

	cfg := config.ForLeanHelixConsensusTests(testKeys.EcdsaSecp256K1KeyPairForTests(0), h.auditBlocksYoungerThan)
	h.instanceId = leanhelixconsensus.CalcInstanceId(cfg.NetworkType(), cfg.VirtualChainId())

	signer, err := signer.New(cfg)
	require.NoError(tb, err)

	h.consensus = leanhelixconsensus.NewLeanHelixConsensusAlgo(ctx, h.gossip, h.blockStorage, h.consensusContext, signer, logger, cfg, registry)
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

func (h *harness) expectConsensusContextRequestOrderingCommitteeNotCalled() {
	h.consensusContext.When("RequestOrderingCommittee", mock.Any, mock.Any).Return(nil, nil).Times(0)
}

func (h *harness) expectConsensusContextRequestOrderingCommittee(leaderNodeIndex int, times int) {
	h.consensusContext.When("RequestOrderingCommittee", mock.Any, mock.Any).Return(&services.RequestCommitteeOutput{
		NodeAddresses: h.getCommitteeWithNodeIndexAsLeader(leaderNodeIndex),
	}, nil).Times(times)
}

func (h *harness) expectConsensusContextRequestNewBlockNotCalled() {
	h.consensusContext.Never("RequestNewTransactionsBlock", mock.Any, mock.Any)
	h.consensusContext.Never("RequestNewResultsBlock", mock.Any, mock.Any)
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
