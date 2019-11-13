// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/lean-helix-go/services/interfaces"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/signer"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/leanhelixconsensus"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const NETWORK_SIZE = 4

const DEFAULT_AUDIT_BLOCKS_YOUNGER_THAN = 0
const DEFAULT_BASE_CONSENSUS_ROUND_TIMEOUT = time.Hour

type harness struct {
	consensus                 *leanhelixconsensus.Service
	gossip                    *gossiptopics.MockLeanHelix
	blockStorage              *services.MockBlockStorage
	consensusContext          *services.MockConsensusContext
	instanceId                lhprimitives.InstanceId
	auditBlocksYoungerThan    time.Duration
	baseConsensusRoundTimeout time.Duration
	metricRegistry            metric.Registry
	logger                    log.Logger
	T                         testing.TB
}

type metrics struct {
	timeSinceLastCommitMillis   *metric.Histogram
	timeSinceLastElectionMillis *metric.Histogram
	currentLeaderMemberId       *metric.Text
	currentElectionCount        *metric.Gauge
	lastCommittedTime           *metric.Gauge
}

func newLeanHelixServiceHarness() *harness {
	h := &harness{
		gossip:                    &gossiptopics.MockLeanHelix{},
		blockStorage:              &services.MockBlockStorage{},
		consensusContext:          &services.MockConsensusContext{},
		auditBlocksYoungerThan:    DEFAULT_AUDIT_BLOCKS_YOUNGER_THAN,
		baseConsensusRoundTimeout: DEFAULT_BASE_CONSENSUS_ROUND_TIMEOUT,
		metricRegistry:            metric.NewRegistry(),
	}

	h.resetAndApplyMockDefaults()

	return h
}

func (h *harness) withAuditBlocksYoungerThan(d time.Duration) *harness {
	h.auditBlocksYoungerThan = d
	return h
}

func (h *harness) withBaseConsensusRoundTimeout(d time.Duration) *harness {
	h.baseConsensusRoundTimeout = d
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
	cfg := config.ForLeanHelixConsensusTests(testKeys.EcdsaSecp256K1KeyPairForTests(0), h.auditBlocksYoungerThan, h.baseConsensusRoundTimeout)
	h.instanceId = leanhelixconsensus.CalcInstanceId(cfg.NetworkType(), cfg.VirtualChainId())
	h.logger = parent.Logger
	h.T = parent.T

	sgnr, err := signer.New(cfg)
	require.NoError(h.T, err)

	h.consensus = leanhelixconsensus.NewLeanHelixConsensusAlgo(ctx, h.gossip, h.blockStorage, h.consensusContext, sgnr, parent.Logger, cfg, h.metricRegistry)
	parent.Supervise(h.consensus)
	return h
}

func (h *harness) getMetrics() *metrics {
	return &metrics{
		timeSinceLastCommitMillis:   h.metricRegistry.Get("ConsensusAlgo.LeanHelix.TimeSinceLastCommit.Millis").(*metric.Histogram),
		timeSinceLastElectionMillis: h.metricRegistry.Get("ConsensusAlgo.LeanHelix.TimeSinceLastElection.Millis").(*metric.Histogram),
		currentElectionCount:        h.metricRegistry.Get("ConsensusAlgo.LeanHelix.CurrentElection.Number").(*metric.Gauge),
		currentLeaderMemberId:       h.metricRegistry.Get("ConsensusAlgo.LeanHelix.CurrentLeaderMemberId.Number").(*metric.Text),
		lastCommittedTime:           h.metricRegistry.Get("ConsensusAlgo.LeanHelix.LastCommitted.TimeNano").(*metric.Gauge),
	}
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

func (h *harness) dontBeFirstInCommitee() {
	h.expectConsensusContextRequestOrderingCommittee((h.myNodeIndex() + 1) % NETWORK_SIZE)
}

func (h *harness) beFirstInCommittee() {
	h.expectConsensusContextRequestOrderingCommittee(h.myNodeIndex())
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

func (h *harness) expectValidateTransactionBlock() {
	h.consensusContext.When("ValidateTransactionsBlock", mock.Any, mock.Any).Return(&services.ValidateTransactionsBlockOutput{})
}

func (h *harness) expectValidateResultsBlock() {
	h.consensusContext.When("ValidateResultsBlock", mock.Any, mock.Any).Return(&services.ValidateResultsBlockOutput{})
}

func (h *harness) expectCommitBlock() {
	h.blockStorage.When("CommitBlock", mock.Any, mock.Any).Return(&services.CommitBlockOutput{})
}

func (h *harness) handleBlockSync(ctx context.Context, blockHeight primitives.BlockHeight) {
	blockPair := builders.BlockPair().WithHeight(blockHeight).WithEmptyLeanHelixBlockProof().Build()

	_, err := h.consensus.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
		Mode:                   handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY,
		BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
		BlockPair:              blockPair,
		PrevCommittedBlockPair: nil,
	})
	require.NoError(h.T, err, "expected HandleBlockConsensus to succeed")
	require.NoError(h.T, test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT, h.consensusContext))
}

func (h *harness) handlePreprepareMessage(ctx context.Context, blockPair *protocol.BlockPairContainer, blockHeight primitives.BlockHeight, view lhprimitives.View, fromNodeInd int) {
	block := leanhelixconsensus.ToLeanHelixBlock(blockPair)
	prpr := generatePreprepareMessage(h.instanceId, block, uint64(blockHeight), view, testKeys.NodeAddressesForTests()[fromNodeInd], h.keyManagerForNode(fromNodeInd))
	_, err := h.consensus.HandleLeanHelixMessage(ctx, &gossiptopics.LeanHelixInput{
		Message: &gossipmessages.LeanHelixMessage{
			Content:   prpr.Content,
			BlockPair: blockPair,
		},
	})
	require.NoError(h.T, err, "expected message to be handled successfully")
}

func (h *harness) handleCommitMessage(ctx context.Context, blockPair *protocol.BlockPairContainer, blockHeight primitives.BlockHeight, view lhprimitives.View, randomSeed uint64, fromNodeInd int) *interfaces.CommitMessage {
	block := leanhelixconsensus.ToLeanHelixBlock(blockPair)
	c := generateCommitMessage(h.instanceId, block, uint64(blockHeight), view, testKeys.NodeAddressesForTests()[fromNodeInd], randomSeed, h.keyManagerForNode(fromNodeInd))
	_, err := h.consensus.HandleLeanHelixMessage(ctx, &gossiptopics.LeanHelixInput{
		Message: &gossipmessages.LeanHelixMessage{
			Content:   c.Content,
			BlockPair: blockPair,
		},
	})
	require.NoError(h.T, err, "expected message to be handled successfully")
	return interfaces.ToConsensusMessage(c).(*interfaces.CommitMessage)
}

func (h *harness) requestOrderingCommittee(ctx context.Context) *services.RequestCommitteeOutput {
	out, err := h.consensusContext.RequestOrderingCommittee(ctx, &services.RequestCommitteeInput{
		CurrentBlockHeight: 0,
		RandomSeed:         0,
		MaxCommitteeSize:   0,
	})
	require.NoError(h.T, err, "expected request ordering committee to succeed")
	return out
}

func (h *harness) networkSize() int {
	return NETWORK_SIZE
}

func (h *harness) myNodeIndex() int {
	return 0
}

func (h *harness) keyManagerForNode(nodeIndex int) interfaces.KeyManager {
	cfg := config.ForLeanHelixConsensusTests(testKeys.EcdsaSecp256K1KeyPairForTests(nodeIndex), h.auditBlocksYoungerThan, h.baseConsensusRoundTimeout)
	sgnr, err := signer.New(cfg)
	require.NoError(h.T, err)
	return leanhelixconsensus.NewKeyManager(h.logger, sgnr)
}
