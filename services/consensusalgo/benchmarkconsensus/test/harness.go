package test

import (
	"context"
	"errors"
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/benchmarkconsensus"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	testInstrumentation "github.com/orbs-network/orbs-network-go/test/harness/instrumentation"
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
	reporting        instrumentation.Reporting
	config           benchmarkconsensus.Config
	service          services.ConsensusAlgoBenchmark
}

func newHarness(
	isLeader bool,
) *harness {

	leaderPublicKey := []byte{0x01}
	nodePublicKey := leaderPublicKey
	if !isLeader {
		nodePublicKey = []byte{0x02}
	}

	config := config.NewHardCodedConfig(
		5,
		nodePublicKey,
		leaderPublicKey,
		consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS,
		5,
	)

	log := testInstrumentation.NewBufferedLog("BenchmarkConsensus")

	gossip := &gossiptopics.MockBenchmarkConsensus{}
	gossip.When("RegisterBenchmarkConsensusHandler", mock.Any).Return().Times(1)

	blockStorage := &services.MockBlockStorage{}
	blockStorage.When("RegisterConsensusBlocksHandler", mock.Any).Return().Times(1)

	consensusContext := &services.MockConsensusContext{}

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

func (h *harness) expectNewBlockProposalRequested(expectedBlockHeight primitives.BlockHeight) {
	txRequestMatcher := func(i interface{}) bool {
		input, ok := i.(*services.RequestNewTransactionsBlockInput)
		return ok && input.BlockHeight.Equal(expectedBlockHeight)
	}
	rxRequestMatcher := func(i interface{}) bool {
		input, ok := i.(*services.RequestNewResultsBlockInput)
		return ok && input.BlockHeight.Equal(expectedBlockHeight)
	}

	builtBlockForReturn := builders.BenchmarkConsensusBlockPair().WithHeight(expectedBlockHeight).Build()
	txReturn := &services.RequestNewTransactionsBlockOutput{
		TransactionsBlock: builtBlockForReturn.TransactionsBlock,
	}
	rxReturn := &services.RequestNewResultsBlockOutput{
		ResultsBlock: builtBlockForReturn.ResultsBlock,
	}

	h.consensusContext.Reset().When("RequestNewTransactionsBlock", mock.AnyIf(fmt.Sprintf("BlockHeight equals %d", expectedBlockHeight), txRequestMatcher)).Return(txReturn, nil).AtLeast(1)
	h.consensusContext.When("RequestNewResultsBlock", mock.AnyIf(fmt.Sprintf("BlockHeight equals %d", expectedBlockHeight), rxRequestMatcher)).Return(rxReturn, nil).AtLeast(1)
}

func (h *harness) verifyNewBlockProposalRequested(t *testing.T) {
	err := test.EventuallyVerify(h.consensusContext)
	if err != nil {
		t.Fatal("Did not create block with ConsensusContext:", err)
	}
}

func (h *harness) expectNewBlockProposalRequestedToFail() {
	h.consensusContext.Reset().When("RequestNewTransactionsBlock", mock.Any).Return(nil, errors.New("consensusContext error")).AtLeast(1)
	h.consensusContext.When("RequestNewResultsBlock", mock.Any).Return(nil, errors.New("consensusContext error")).Times(0)
}

func (h *harness) expectNewBlockProposalNotRequested() {
	h.consensusContext.Reset().When("RequestNewTransactionsBlock", mock.Any).Return(nil, nil).Times(0)
	h.consensusContext.When("RequestNewResultsBlock", mock.Any).Return(nil, nil).Times(0)
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

func (h *harness) receivedCommittedViaGossip(message *gossipmessages.BenchmarkConsensusCommittedMessage) {
	h.service.HandleBenchmarkConsensusCommitted(&gossiptopics.BenchmarkConsensusCommittedInput{
		RecipientPublicKey: nil,
		Message:            message,
	})
}

func (h *harness) receivedCommittedViaGossipFromSeveral(numNodes int, lastCommitted primitives.BlockHeight, validSignature bool) {
	aCommitted := builders.BenchmarkConsensusCommittedMessage().WithLastCommittedHeight(lastCommitted)
	for i := 0; i < numNodes; i++ {
		var c *gossipmessages.BenchmarkConsensusCommittedMessage
		if validSignature {
			c = aCommitted.WithSenderSignature(nil, []byte{byte(i + 5)}).Build()
		} else {
			c = aCommitted.WithInvalidSenderSignature(nil, []byte{byte(i + 5)}).Build()
		}
		h.receivedCommittedViaGossip(c)
	}
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

func (h *harness) expectCommitSaveAndReply(expectedBlockPair *protocol.BlockPairContainer, expectedLastCommitted primitives.BlockHeight, expectedRecipient primitives.Ed25519PublicKey, expectedSender primitives.Ed25519PublicKey) {
	lastCommittedReplyMatcher := func(i interface{}) bool {
		input, ok := i.(*gossiptopics.BenchmarkConsensusCommittedInput)
		return ok &&
			input.Message.Status.LastCommittedBlockHeight() == expectedLastCommitted &&
			input.RecipientPublicKey.Equal(expectedRecipient) &&
			input.Message.Sender.SenderPublicKey().Equal(expectedSender)
	}

	h.blockStorage.Reset().When("CommitBlock", &services.CommitBlockInput{expectedBlockPair}).Return(nil, nil).Times(1)
	h.gossip.Reset().When("SendBenchmarkConsensusCommitted", mock.AnyIf(fmt.Sprintf("LastCommittedBlockHeight equals %d, recipient equals %s and sender equals %s", expectedLastCommitted, expectedRecipient, expectedSender), lastCommittedReplyMatcher)).Times(1)
}

func (h *harness) verifyCommitSaveAndReply(t *testing.T) {
	err := test.EventuallyVerify(h.blockStorage, h.gossip)
	if err != nil {
		t.Fatal("Did not commit and reply to block:", err)
	}
}

func (h *harness) expectCommitSent(expectedBlockHeight primitives.BlockHeight, expectedSender primitives.Ed25519PublicKey) {
	commitSentMatcher := func(i interface{}) bool {
		input, ok := i.(*gossiptopics.BenchmarkConsensusCommitInput)
		return ok &&
			input.Message.BlockPair.TransactionsBlock.Header.BlockHeight().Equal(expectedBlockHeight) &&
			input.Message.BlockPair.ResultsBlock.Header.BlockHeight().Equal(expectedBlockHeight) &&
			input.Message.BlockPair.ResultsBlock.BlockProof.IsTypeBenchmarkConsensus() &&
			input.Message.BlockPair.ResultsBlock.BlockProof.BenchmarkConsensus().Sender().SenderPublicKey().Equal(expectedSender)
	}

	h.gossip.ResetAndWhen("BroadcastBenchmarkConsensusCommit", mock.AnyIf(fmt.Sprintf("BlockHeight equals %d, block proof is BenchmarkConsensus and sender equals %s", expectedBlockHeight, expectedSender), commitSentMatcher)).AtLeast(1)
	h.gossip.When("RegisterBenchmarkConsensusHandler", mock.Any).Return()
}

func (h *harness) verifyCommitSent(t *testing.T) {
	err := test.EventuallyVerify(h.gossip)
	if err != nil {
		t.Fatal("Did not broadcast block commit:", err)
	}
}

func (h *harness) expectCommitNotSent() {
	h.gossip.ResetAndWhen("BroadcastBenchmarkConsensusCommit", mock.Any).Times(0)
	h.gossip.When("RegisterBenchmarkConsensusHandler", mock.Any).Return()
}

func (h *harness) verifyCommitNotSent(t *testing.T) {
	err := test.ConsistentlyVerify(h.gossip)
	if err != nil {
		t.Fatal("Did broadcast block commit:", err)
	}
}
