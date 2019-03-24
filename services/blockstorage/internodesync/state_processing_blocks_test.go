// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package internodesync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStateProcessingBlocks_CommitsAccordinglyAndMovesToCollectingAvailabilityResponses(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness(log.DefaultTestingLogger(t))

		message := builders.BlockSyncResponseInput().
			WithFirstBlockHeight(10).
			WithLastBlockHeight(20).
			WithLastCommittedBlockHeight(20).
			Build().Message

		h.expectBlockValidationQueriesFromStorage(11)
		h.expectBlockCommitsToStorage(11)

		state := h.factory.CreateProcessingBlocksState(message)
		nextState := state.processState(ctx)

		require.IsType(t, &collectingAvailabilityResponsesState{}, nextState, "next state after commit should be collecting availability responses")
		h.verifyMocks(t)
	})
}

func TestStateProcessingBlocks_ReturnsToIdleWhenNoBlocksReceived(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness(log.DefaultTestingLogger(t))

		state := h.factory.CreateProcessingBlocksState(nil)
		nextState := state.processState(ctx)

		require.IsType(t, &idleState{}, nextState, "commit initialized invalid should move to idle")
	})
}

func TestStateProcessingBlocks_ValidateBlockFailureReturnsToCollectingAvailabilityResponses(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness(log.DefaultTestingLoggerAllowingErrors(t, "failed to validate block received via sync"))

		message := builders.BlockSyncResponseInput().
			WithFirstBlockHeight(10).
			WithLastBlockHeight(20).
			WithLastCommittedBlockHeight(20).
			Build().Message

		h.expectBlockValidationQueriesFromStorageAndFailLastValidation(11, message.SignedChunkRange.FirstBlockHeight())
		h.expectBlockCommitsToStorage(10)

		state := h.factory.CreateProcessingBlocksState(message)
		nextState := state.processState(ctx)

		require.IsType(t, &collectingAvailabilityResponsesState{}, nextState, "next state after validation error should be collecting availability responses")
		h.verifyMocks(t)
	})
}

func TestStateProcessingBlocks_CommitBlockFailureReturnsToCollectingAvailabilityResponses(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness(log.DefaultTestingLoggerAllowingErrors(t, "failed to commit block received via sync"))

		message := builders.BlockSyncResponseInput().
			WithFirstBlockHeight(10).
			WithLastBlockHeight(20).
			WithLastCommittedBlockHeight(20).
			Build().Message

		h.expectBlockValidationQueriesFromStorage(11)
		h.expectBlockCommitsToStorageAndFailLastCommit(11, message.SignedChunkRange.FirstBlockHeight())

		processingState := h.factory.CreateProcessingBlocksState(message)
		next := processingState.processState(ctx)

		require.IsType(t, &collectingAvailabilityResponsesState{}, next, "next state after commit error should be collecting availability responses")

		h.verifyMocks(t)
	})
}

func TestStateProcessingBlocks_TerminatesOnContextTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h := newBlockSyncHarness(log.DefaultTestingLogger(t))

	message := builders.BlockSyncResponseInput().
		WithFirstBlockHeight(10).
		WithLastBlockHeight(20).
		WithLastCommittedBlockHeight(20).
		Build().Message

	cancel()
	state := h.factory.CreateProcessingBlocksState(message)
	nextState := state.processState(ctx)

	require.Nil(t, nextState, "next state should be nil on context termination")
}
