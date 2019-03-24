// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"testing"
)

func (h *harness) receivedCommitViaGossip(ctx context.Context, blockPair *protocol.BlockPairContainer) {
	h.service.HandleBenchmarkConsensusCommit(ctx, &gossiptopics.BenchmarkConsensusCommitInput{
		Message: &gossipmessages.BenchmarkConsensusCommitMessage{
			BlockPair: blockPair,
		},
	})
}

// expectations

func (h *harness) expectCommitIgnored() {
	h.blockStorage.When("CommitBlock", mock.Any, mock.Any).Return(nil, nil).Times(0)
	h.gossip.When("SendBenchmarkConsensusCommitted", mock.Any, mock.Any).Return(nil, nil).Times(0)
}

func (h *harness) verifyCommitIgnored(t *testing.T) {
	err := test.ConsistentlyVerify(test.CONSISTENTLY_ACCEPTANCE_TIMEOUT, h.blockStorage, h.gossip)
	if err != nil {
		t.Fatal("Did not ignore block:", err)
	}
}

func (h *harness) expectCommitSaveAndReply(expectedBlockPair *protocol.BlockPairContainer, expectedLastCommitted primitives.BlockHeight, expectedRecipient primitives.NodeAddress, expectedSender primitives.NodeAddress) {
	lastCommittedReplyMatcher := func(i interface{}) bool {
		input, ok := i.(*gossiptopics.BenchmarkConsensusCommittedInput)
		return ok &&
			input.Message.Status.LastCommittedBlockHeight() == expectedLastCommitted &&
			input.RecipientNodeAddress.Equal(expectedRecipient) &&
			input.Message.Sender.SenderNodeAddress().Equal(expectedSender)
	}

	h.blockStorage.When("CommitBlock", mock.Any, &services.CommitBlockInput{BlockPair: expectedBlockPair}).Return(nil, nil).Times(1)
	h.gossip.When("SendBenchmarkConsensusCommitted", mock.Any, mock.AnyIf(fmt.Sprintf("LastCommittedBlockHeight equals %d, recipient equals %s and sender equals %s", expectedLastCommitted, expectedRecipient, expectedSender), lastCommittedReplyMatcher)).Times(1)
}

func (h *harness) verifyCommitSaveAndReply(t *testing.T) {
	err := test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT, h.blockStorage, h.gossip)
	if err != nil {
		t.Fatal("Did not commit and reply to block:", err)
	}
}

func (h *harness) expectCommitReplyWithoutSave(expectedBlockPair *protocol.BlockPairContainer, expectedLastCommitted primitives.BlockHeight, expectedRecipient primitives.NodeAddress, expectedSender primitives.NodeAddress) {
	lastCommittedReplyMatcher := func(i interface{}) bool {
		input, ok := i.(*gossiptopics.BenchmarkConsensusCommittedInput)
		return ok &&
			input.Message.Status.LastCommittedBlockHeight() == expectedLastCommitted &&
			input.RecipientNodeAddress.Equal(expectedRecipient) &&
			input.Message.Sender.SenderNodeAddress().Equal(expectedSender)
	}

	h.blockStorage.When("CommitBlock", mock.Any, &services.CommitBlockInput{BlockPair: expectedBlockPair}).Return(nil, nil).Times(0)
	h.gossip.When("SendBenchmarkConsensusCommitted", mock.Any, mock.AnyIf(fmt.Sprintf("LastCommittedBlockHeight equals %d, recipient equals %s and sender equals %s", expectedLastCommitted, expectedRecipient, expectedSender), lastCommittedReplyMatcher)).Times(1)
}

func (h *harness) verifyCommitReplyWithoutSave(t *testing.T) {
	err := test.ConsistentlyVerify(test.CONSISTENTLY_ACCEPTANCE_TIMEOUT, h.blockStorage)
	if err != nil {
		t.Fatal("Did save the block to block storage:", err)
	}
	err = test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT, h.gossip)
	if err != nil {
		t.Fatal("Did not reply to block:", err)
	}
}

func (h *harness) expectCommitBroadcastViaGossip(expectedBlockHeight primitives.BlockHeight, expectedSender primitives.NodeAddress) {
	commitSentMatcher := func(i interface{}) bool {
		input, ok := i.(*gossiptopics.BenchmarkConsensusCommitInput)
		return ok &&
			input.Message.BlockPair.TransactionsBlock.Header.BlockHeight().Equal(expectedBlockHeight) &&
			input.Message.BlockPair.ResultsBlock.Header.BlockHeight().Equal(expectedBlockHeight) &&
			input.Message.BlockPair.ResultsBlock.BlockProof.IsTypeBenchmarkConsensus() &&
			input.Message.BlockPair.ResultsBlock.BlockProof.BenchmarkConsensus().NodesIterator().NextNodes().SenderNodeAddress().Equal(expectedSender)
	}

	h.gossip.When("BroadcastBenchmarkConsensusCommit", mock.Any, mock.AnyIf(fmt.Sprintf("BlockHeight equals %d, block proof is BenchmarkConsensus and sender equals %s", expectedBlockHeight, expectedSender), commitSentMatcher)).AtLeast(1)
}

func (h *harness) verifyCommitBroadcastViaGossip(t *testing.T) {
	err := test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT, h.gossip)
	if err != nil {
		t.Fatal("Did not broadcast block commit:", err)
	}
}

func (h *harness) expectCommitNotSent() {
	h.gossip.When("BroadcastBenchmarkConsensusCommit", mock.Any, mock.Any).Times(0)
}

func (h *harness) verifyCommitNotSent(t *testing.T) {
	err := test.ConsistentlyVerify(test.CONSISTENTLY_ACCEPTANCE_TIMEOUT, h.gossip)
	if err != nil {
		t.Fatal("Did broadcast block commit:", err)
	}
}
