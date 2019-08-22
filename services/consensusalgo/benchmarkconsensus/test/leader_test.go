// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"testing"
)

func newLeaderHarnessWaitingForCommittedMessages(parent *test.ConcurrencyHarness, ctx context.Context) *harness {
	h := newHarness(parent, true)
	h.expectNewBlockProposalNotRequested()
	h.expectCommitBroadcastViaGossip(0, h.config.NodeAddress())
	h.createService(ctx)
	h.verifyCommitBroadcastViaGossip(parent.T)
	return h
}

func TestLeaderInit(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, parent *test.ConcurrencyHarness) {
		h := newLeaderHarnessWaitingForCommittedMessages(parent, ctx)

		h.verifyHandlerRegistrations(t)
		h.verifyNewBlockProposalNotRequested(t)
	})
}

// this test protects against a rare race where we loaded blocks from storage and node sync didn't update followers before leader started
// see https://circleci.com/gh/orbs-network/orbs-network-go/17275#tests/containers/2
func TestLeaderInitWithExistingBlocks_DoesNotCreateGenesisBlock(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, parent *test.ConcurrencyHarness) {
		block17 := builders.BlockPair().WithHeight(17).Build()

		h := newHarness(parent, true)
		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(17, h.config.NodeAddress())

		// resetting because go-mock doesn't register later When() calls as taking precedence over older calls, and newHarness() stubs this same method with a no-op
		h.blockStorage.Reset().When("RegisterConsensusBlocksHandler", mock.Any).Call(func(handler handlers.ConsensusBlocksHandler) {
			// this recreates how block storage updates us on last committed block on init (via the call to RegisterConsensusBlocksHandler)
			_, err := handler.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
				Mode:                   handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY,
				BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
				BlockPair:              block17,
				PrevCommittedBlockPair: nil,
			})
			require.NoError(t, err, "failed calling HandleBlockConsensus")

		}).Times(1)

		h.createService(ctx)

		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderCommitsConsecutiveBlocksAfterEnoughConfirmations(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, parent *test.ConcurrencyHarness) {
		h := newLeaderHarnessWaitingForCommittedMessages(parent, ctx)

		t.Log("Nodes confirmed height 0 (genesis), commit height 1")

		c0 := multipleCommittedMessages().WithHeight(0).WithCountAboveQuorum(h.config).Build()
		h.expectNewBlockProposalRequestedAndSaved(1)
		h.expectCommitBroadcastViaGossip(1, h.config.NodeAddress())

		h.receivedCommittedMessagesViaGossip(ctx, c0)
		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)

		t.Log("Nodes confirmed height 1, commit height 2")

		c1 := multipleCommittedMessages().WithHeight(1).WithCountAboveQuorum(h.config).Build()
		h.expectNewBlockProposalRequestedAndSaved(2)
		h.expectCommitBroadcastViaGossip(2, h.config.NodeAddress())

		h.receivedCommittedMessagesViaGossip(ctx, c1)
		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderRetriesCommitOnErrorGeneratingBlock(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, parent *test.ConcurrencyHarness) {
		h := newLeaderHarnessWaitingForCommittedMessages(parent, ctx)

		t.Log("Nodes confirmed height 0 (genesis), fail to generate height 1")

		c0 := multipleCommittedMessages().WithHeight(0).WithCountAboveQuorum(h.config).Build()
		h.expectNewBlockProposalRequestedToFail()
		h.expectCommitNotSent()

		h.receivedCommittedMessagesViaGossip(ctx, c0)
		h.verifyNewBlockProposalRequestedAndNotSaved(t)
		h.verifyCommitNotSent(t)

		t.Log("Stop failing to generate, commit height 1")

		h.expectNewBlockProposalRequestedAndSaved(1)
		h.expectCommitBroadcastViaGossip(1, h.config.NodeAddress())

		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderRetriesCommitAfterNotEnoughConfirmations(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, parent *test.ConcurrencyHarness) {
		h := newLeaderHarnessWaitingForCommittedMessages(parent, ctx)

		t.Log("Nodes confirmed height 0 (genesis), commit height 1")

		c0 := multipleCommittedMessages().WithHeight(0).WithCountAboveQuorum(h.config).Build()
		h.expectNewBlockProposalRequestedAndSaved(1)
		h.expectCommitBroadcastViaGossip(1, h.config.NodeAddress())

		h.receivedCommittedMessagesViaGossip(ctx, c0)
		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)

		t.Log("Not enough nodes confirmed height 1, commit height 1 again")

		c1 := multipleCommittedMessages().WithHeight(1).WithCountBelowQuorum(h.config).Build()
		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(1, h.config.NodeAddress())

		h.receivedCommittedMessagesViaGossip(ctx, c1)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresBadCommittedMessageSignatures(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, parent *test.ConcurrencyHarness) {
		h := newLeaderHarnessWaitingForCommittedMessages(parent, ctx)

		t.Log("Bad signatures nodes confirmed height 0 (genesis), commit height 0 again")

		c0 := multipleCommittedMessages().WithHeight(0).WithInvalidSignatures().WithCountAboveQuorum(h.config).Build()
		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(0, h.config.NodeAddress())

		h.receivedCommittedMessagesViaGossip(ctx, c0)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresNonValidatorSigners(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, parent *test.ConcurrencyHarness) {
		h := newLeaderHarnessWaitingForCommittedMessages(parent, ctx)

		t.Log("Non validator nodes confirmed height 0 (genesis), commit height 0 again")

		c0 := multipleCommittedMessages().WithHeight(0).FromNonGenesisValidators().WithCountAboveQuorum(h.config).Build()
		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(0, h.config.NodeAddress())

		h.receivedCommittedMessagesViaGossip(ctx, c0)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresOldConfirmations(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, parent *test.ConcurrencyHarness) {
		h := newLeaderHarnessWaitingForCommittedMessages(parent, ctx)

		t.Log("Nodes confirmed height 0 (genesis), commit height 1")

		c0 := multipleCommittedMessages().WithHeight(0).WithCountAboveQuorum(h.config).Build()
		h.expectNewBlockProposalRequestedAndSaved(1)
		h.expectCommitBroadcastViaGossip(1, h.config.NodeAddress())

		h.receivedCommittedMessagesViaGossip(ctx, c0)
		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)

		t.Log("Nodes confirmed height 0 (genesis) again, commit height 1 again")

		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(1, h.config.NodeAddress())

		h.receivedCommittedMessagesViaGossip(ctx, c0)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresFutureConfirmations(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, parent *test.ConcurrencyHarness) {
		h := newLeaderHarnessWaitingForCommittedMessages(parent, ctx)

		t.Log("Nodes confirmed height 1000, commit height 0 (genesis) again")

		c1000 := multipleCommittedMessages().WithHeight(1000).WithCountAboveQuorum(h.config).Build()
		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(0, h.config.NodeAddress())

		h.receivedCommittedMessagesViaGossip(ctx, c1000)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}
