package test

import (
	"context"
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/benchmarkconsensus"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/crypto"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"testing"
)

type harness struct {
	gossip           *gossiptopics.MockBenchmarkConsensus
	blockStorage     *services.MockBlockStorage
	consensusContext *services.MockConsensusContext
	reporting        instrumentation.BasicLogger
	config           benchmarkconsensus.Config
	service          services.ConsensusAlgoBenchmark
}

func newHarness(
	isLeader bool,
) *harness {

	leaderPublicKey := crypto.Ed25519KeyPairForTests(1).PublicKey()
	nodePublicKey := leaderPublicKey
	if !isLeader {
		nodePublicKey = []byte{0x02}
	}

	config := config.NewHardCodedConfig(
		5,
		nodePublicKey,
		leaderPublicKey,
		consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS,
		70,
	)

	log := instrumentation.GetLogger()

	gossip := &gossiptopics.MockBenchmarkConsensus{}
	gossip.When("RegisterBenchmarkConsensusHandler", mock.Any).Return()

	blockStorage := &services.MockBlockStorage{}
	blockStorage.When("RegisterConsensusBlocksHandler", mock.Any).Return()

	consensusContext := &services.MockConsensusContext{}
	if isLeader {
		consensusContext.When("RequestNewTransactionsBlock", mock.Any).Return(nil, nil)
	}

	return &harness{
		gossip:           gossip,
		blockStorage:     blockStorage,
		consensusContext: consensusContext,
		reporting:        log,
		config:           config,
		service:          nil,
	}
}

func (h *harness) createService(ctx context.Context) {
	h.service = benchmarkconsensus.NewBenchmarkConsensusAlgo(
		ctx,
		h.gossip,
		h.blockStorage,
		h.consensusContext,
		h.reporting,
		h.config,
	)
}

func nonLeaderPublicKey() primitives.Ed25519PublicKey {
	return []byte{0x99}
}

func (h *harness) expectHandlerRegistrations() {
	h.gossip.Reset().When("RegisterBenchmarkConsensusHandler", mock.Any).Return().Times(1)
	h.blockStorage.Reset().When("RegisterConsensusBlocksHandler", mock.Any).Return().Times(1)
}

func (h *harness) verifyHandlerRegistrations(t *testing.T) {
	ok, err := h.gossip.Verify()
	if !ok {
		t.Fatal("Did not register with Gossip:", err)
	}
	ok, err = h.blockStorage.Verify()
	if !ok {
		t.Fatal("Did not register with BlockStorage:", err)
	}
}

func (h *harness) expectNewBlockProposalRequested() {
	h.consensusContext.Reset().When("RequestNewTransactionsBlock", mock.Any).Return(nil, nil).AtLeast(1)
}

func (h *harness) verifyNewBlockProposalRequested(t *testing.T) {
	err := test.EventuallyVerify(h.consensusContext)
	if err != nil {
		t.Fatal("Did not create block with ConsensusContext:", err)
	}
}

func (h *harness) expectNewBlockProposalNotRequested() {
	h.consensusContext.Reset().When("RequestNewTransactionsBlock", mock.Any).Return(nil, nil).Times(0)
}

func (h *harness) verifyNewBlockProposalNotRequested(t *testing.T) {
	err := test.ConsistentlyVerify(h.consensusContext)
	if err != nil {
		t.Fatal("Did create block with ConsensusContext:", err)
	}
}

func (h *harness) receivedCommitViaGossip(blockPair *protocol.BlockPairContainer) {
	h.service.HandleBenchmarkConsensusCommit(&gossiptopics.BenchmarkConsensusCommitInput{
		Message: &gossipmessages.BenchmarkConsensusCommitMessage{
			BlockPair: blockPair,
		},
	})
}

func (h *harness) expectCommitIgnored() {
	h.blockStorage.Reset().When("CommitBlock", mock.Any).Return(nil, nil).Times(0)
	h.gossip.Reset().When("SendBenchmarkConsensusCommitted", mock.Any).Return(nil, nil).Times(0)
}

func (h *harness) verifyCommitIgnored(t *testing.T) {
	err := test.ConsistentlyVerify(h.blockStorage, h.gossip)
	if err != nil {
		t.Fatal("Did not ignore block:", err)
	}
}

func (h *harness) expectCommitSaveAndReply(expectedBlockPair *protocol.BlockPairContainer, expectedLastCommitted primitives.BlockHeight) {
	lastCommittedReplyMatcher := func(i interface{}) bool {
		input, ok := i.(*gossiptopics.BenchmarkConsensusCommittedInput)
		return ok && input.Message.Status.LastCommittedBlockHeight() == expectedLastCommitted
	}

	h.blockStorage.Reset().When("CommitBlock", &services.CommitBlockInput{expectedBlockPair}).Return(nil, nil).Times(1)
	h.gossip.Reset().When("SendBenchmarkConsensusCommitted", mock.AnyIf(fmt.Sprintf("Message.Status.LastCommittedBlockHeight of %d", expectedLastCommitted), lastCommittedReplyMatcher)).Times(1)
}

func (h *harness) verifyCommitSaveAndReply(t *testing.T) {
	err := test.EventuallyVerify(h.blockStorage, h.gossip)
	if err != nil {
		t.Fatal("Did not commit and reply to block:", err)
	}
}
