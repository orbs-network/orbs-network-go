// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package internodesync

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestStateProcessingBlocksAscending_CommitsAccordinglyAndMovesToCollectingAvailabilityResponses(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			h := newBlockSyncHarness(harness.Logger).
				withDescendingEnabled(false)
			message := builders.BlockSyncResponseInput().
				WithBlocksOrder(gossipmessages.SYNC_BLOCKS_ORDER_ASCENDING).
				WithFirstBlockHeight(1).
				WithLastBlockHeight(11).
				WithLastCommittedBlockHeight(11).
				Build().Message

			h.expectBlockValidationQueriesFromStorage(11)
			h.expectBlockCommitsToStorage(11)
			state := h.factory.CreateProcessingBlocksState(message)
			nextState := state.processState(ctx)
			require.IsType(t, &collectingAvailabilityResponsesState{}, nextState, "next state after commit should be collecting availability responses")
			h.verifyMocks(t)
		})
	})
}

func TestStateProcessingBlocksDescending_CommitsAccordinglyAndMovesToCollectingAvailabilityResponses(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			h := newBlockSyncHarness(harness.Logger).
				withDescendingEnabled(true)

			// First check
			message := builders.BlockSyncResponseInput().
				WithBlocksOrder(gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING).
				WithFirstBlockHeight(11).
				WithLastBlockHeight(1).
				WithLastCommittedBlockHeight(11).
				Build().Message

			h.expectBlockValidationQueriesFromStorage(11)
			h.expectBlockCommitsToStorage(11)
			state := h.factory.CreateProcessingBlocksState(message)
			h.storage.When("ValidateChainTip", mock.Any, mock.Any).Return(nil, nil).Times(1)
			nextState := state.processState(ctx)
			require.IsType(t, &collectingAvailabilityResponsesState{}, nextState, "next state after commit should be collecting availability responses")

			// Second Check
			message = builders.BlockSyncResponseInput().
				WithBlocksOrder(gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING).
				WithFirstBlockHeight(20).
				WithLastBlockHeight(11).
				WithLastCommittedBlockHeight(20).
				Build().Message

			block := builders.BlockPair().
				WithHeight(primitives.BlockHeight(10)).
				Build()
			syncState := SyncState{InOrderBlock: block, TopBlock: block, LastSyncedBlock: block}
			h.storage.When("GetSyncState").Return(syncState).Times(1)
			h.storage.When("GetBlock", mock.Any).Return(nil).Times(1)
			h.storage.When("ValidateBlockForCommit", mock.Any, mock.Any).Return(nil, nil).Times(10)
			h.storage.When("ValidateChainTip", mock.Any, mock.Any).Return(nil, nil).Times(1)

			h.expectBlockCommitsToStorage(10)

			state = h.factory.CreateProcessingBlocksState(message)
			nextState = state.processState(ctx)
			require.IsType(t, &collectingAvailabilityResponsesState{}, nextState, "next state after commit should be collecting availability responses")

			// Third check
			var blocks []*protocol.BlockPairContainer
			var prevBlock *protocol.BlockPairContainer
			for i := 10; i <= 30; i++ {
				blockPair := builders.BlockPair().
					WithHeight(primitives.BlockHeight(i)).
					WithPrevBlock(prevBlock).
					Build()
				prevBlock = blockPair
				blocks = append(blocks, blockPair)
			}
			reverse(blocks)

			lastSyncedBlock := blocks[9]
			topBlock := blocks[0]
			inOrderBlock := blocks[20]
			syncState = SyncState{InOrderBlock: inOrderBlock, TopBlock: topBlock, LastSyncedBlock: lastSyncedBlock}
			message = builders.BlockSyncResponseInput().
				WithBlocksOrder(gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING).
				WithBlocks(blocks[10:20]).
				WithFirstBlockHeight(20).
				WithLastBlockHeight(11).
				WithLastCommittedBlockHeight(30).
				Build().Message

			h.storage.When("GetSyncState").Return(syncState).Times(1)
			h.storage.When("GetBlock", mock.Any).Return(lastSyncedBlock)
			h.storage.When("ValidateBlockForCommit", mock.Any, mock.Any).Return(nil, nil).Times(10)
			h.expectBlockCommitsToStorage(10)

			state = h.factory.CreateProcessingBlocksState(message)
			nextState = state.processState(ctx)
			require.IsType(t, &collectingAvailabilityResponsesState{}, nextState, "next state after commit should be collecting availability responses")

			h.verifyMocks(t)
		})
	})
}

func TestStateProcessingBlocks_ReturnsToIdleWhenNoBlocksReceived(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			h := newBlockSyncHarness(harness.Logger)
			h.storage.When("GetBlock", mock.Any).Return(nil, nil)
			state := h.factory.CreateProcessingBlocksState(nil)
			nextState := state.processState(ctx)

			require.IsType(t, &idleState{}, nextState, "commit initialized invalid should move to idle")
		})
	})
}

func TestStateProcessingBlocksAscending_ValidateBlockFailureReturnsToCollectingAvailabilityResponses(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			h := newBlockSyncHarness(harness.Logger).
				withDescendingEnabled(false)
			harness.AllowErrorsMatching("failed to validate block received via sync")

			message := builders.BlockSyncResponseInput().
				WithBlocksOrder(gossipmessages.SYNC_BLOCKS_ORDER_ASCENDING).
				WithFirstBlockHeight(1).
				WithLastBlockHeight(11).
				WithLastCommittedBlockHeight(11).
				Build().Message

			expectedFailedBlockHeight := primitives.BlockHeight(11)
			h.expectBlockValidationQueriesFromStorageAndFailLastValidation(11, expectedFailedBlockHeight)
			h.expectBlockCommitsToStorage(10)

			state := h.factory.CreateProcessingBlocksState(message)
			nextState := state.processState(ctx)

			require.IsType(t, &collectingAvailabilityResponsesState{}, nextState, "next state after validation error should be collecting availability responses")
			h.verifyMocks(t)
		})
	})
}

func TestStateProcessingBlocksDescending_ValidateBlockFailureReturnsToCollectingAvailabilityResponses(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			h := newBlockSyncHarness(harness.Logger).
				withDescendingEnabled(true)
			harness.AllowErrorsMatching("failed to validate block received via sync")

			message := builders.BlockSyncResponseInput().
				WithBlocksOrder(gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING).
				WithFirstBlockHeight(11).
				WithLastBlockHeight(1).
				WithLastCommittedBlockHeight(11).
				Build().Message

			expectedFailedBlockHeight := primitives.BlockHeight(1)
			h.storage.When("ValidateChainTip", mock.Any, mock.Any).Return(nil, nil).Times(1)
			h.expectBlockValidationQueriesFromStorageAndFailLastValidation(11, expectedFailedBlockHeight)
			h.expectBlockCommitsToStorage(10)

			state := h.factory.CreateProcessingBlocksState(message)
			nextState := state.processState(ctx)

			require.IsType(t, &collectingAvailabilityResponsesState{}, nextState, "next state after validation error should be collecting availability responses")
			h.verifyMocks(t)
		})
	})
}

func TestStateProcessingBlocksDescending_ValidateChainTipFailureReturnsToCollectingAvailabilityResponses(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			h := newBlockSyncHarness(harness.Logger).
				withDescendingEnabled(true)
			harness.AllowErrorsMatching("failed to validate block received via sync")

			message := builders.BlockSyncResponseInput().
				WithBlocksOrder(gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING).
				WithFirstBlockHeight(11).
				WithLastBlockHeight(1).
				WithLastCommittedBlockHeight(11).
				Build().Message

			h.storage.When("GetSyncState").Return(nil).Times(1)
			h.storage.When("ValidateChainTip", mock.Any, mock.Any).Return(nil, errors.New(" failed to validate the chain tip")).Times(1)
			state := h.factory.CreateProcessingBlocksState(message)
			nextState := state.processState(ctx)

			require.IsType(t, &collectingAvailabilityResponsesState{}, nextState, "next state after validation error should be collecting availability responses")
			h.verifyMocks(t)
		})
	})
}

func TestStateProcessingBlocksDescending_ValidateBlockChunkRangeFailureReturnsToCollectingAvailabilityResponses(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			h := newBlockSyncHarness(harness.Logger).
				withDescendingEnabled(true)
			harness.AllowErrorsMatching("failed to verify the blocks chunk range received via sync")

			blockPair := builders.BlockPair().
				WithHeight(primitives.BlockHeight(10)).
				Build()

			syncState := SyncState{InOrderBlock: blockPair, TopBlock: blockPair, LastSyncedBlock: blockPair}
			h.storage.When("GetSyncState").Return(syncState).Times(1)

			message := builders.BlockSyncResponseInput().
				WithBlocksOrder(gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING).
				WithFirstBlockHeight(9).
				WithLastBlockHeight(1).
				WithLastCommittedBlockHeight(9).
				Build().Message

			state := h.factory.CreateProcessingBlocksState(message)
			nextState := state.processState(ctx)
			require.IsType(t, &collectingAvailabilityResponsesState{}, nextState, "next state after validation error should be collecting availability responses")

			topBlock := builders.BlockPair().
				WithHeight(primitives.BlockHeight(30)).
				Build()

			lastSyncedBlock := builders.BlockPair().
				WithHeight(primitives.BlockHeight(20)).
				Build()

			syncState = SyncState{InOrderBlock: blockPair, TopBlock: topBlock, LastSyncedBlock: lastSyncedBlock}
			h.storage.When("GetSyncState").Return(syncState).Times(1)
			message = builders.BlockSyncResponseInput().
				WithBlocksOrder(gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING).
				WithFirstBlockHeight(12).
				WithLastBlockHeight(11).
				WithLastCommittedBlockHeight(12).
				Build().Message

			state = h.factory.CreateProcessingBlocksState(message)
			nextState = state.processState(ctx)

			require.IsType(t, &collectingAvailabilityResponsesState{}, nextState, "next state after validation error should be collecting availability responses")
			h.verifyMocks(t)
		})
	})
}

func TestStateProcessingBlocksDescending_ValidatePosChainPrevHashFailure(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			h := newBlockSyncHarness(harness.Logger).
				withDescendingEnabled(true)
			harness.AllowErrorsMatching("failed to verify the blocks chunk PoS received via sync")

			inOrderBlock := builders.BlockPair().
				WithHeight(primitives.BlockHeight(10)).
				Build()

			lastSyncedBlock := builders.BlockPair().
				WithHeight(primitives.BlockHeight(20)).
				Build()
			syncState := SyncState{InOrderBlock: inOrderBlock, TopBlock: lastSyncedBlock, LastSyncedBlock: lastSyncedBlock}
			h.storage.When("GetSyncState").Return(syncState).Times(1)
			h.storage.Never("ValidateBlockForCommit", mock.Any, mock.Any)
			h.storage.Never("NodeSyncCommitBlock", mock.Any, mock.Any)

			var blocks []*protocol.BlockPairContainer
			var prevBlock *protocol.BlockPairContainer
			for i := 11; i <= 20; i++ {
				if i == 15 {
					prevBlock = nil
				}
				blockPair := builders.BlockPair().
					WithHeight(primitives.BlockHeight(i)).
					WithPrevBlock(prevBlock).
					Build()
				prevBlock = blockPair
				blocks = append(blocks, blockPair)
			}
			reverse(blocks)
			h.storage.When("GetBlock", mock.Any).Return(blocks[0])

			message := builders.BlockSyncResponseInput().
				WithBlocksOrder(gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING).
				WithBlocks(blocks[1:]).
				WithFirstBlockHeight(19).
				WithLastBlockHeight(11).
				WithLastCommittedBlockHeight(20).
				Build().Message

			state := h.factory.CreateProcessingBlocksState(message)
			nextState := state.processState(ctx)
			require.IsType(t, &collectingAvailabilityResponsesState{}, nextState, "next state after validation error should be collecting availability responses")
			h.verifyMocks(t)
		})
	})
}

func reverse(arr []*protocol.BlockPairContainer) {
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}
}

func TestStateProcessingBlocksAscending_CommitBlockFailureReturnsToCollectingAvailabilityResponses(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			h := newBlockSyncHarness(harness.Logger).
				withDescendingEnabled(false)
			harness.AllowErrorsMatching("failed to commit block received via sync")

			message := builders.BlockSyncResponseInput().
				WithBlocksOrder(gossipmessages.SYNC_BLOCKS_ORDER_ASCENDING).
				WithFirstBlockHeight(1).
				WithLastBlockHeight(11).
				WithLastCommittedBlockHeight(11).
				Build().Message

			h.expectBlockValidationQueriesFromStorage(11)
			h.expectBlockCommitsToStorageAndFailLastCommit(11, message.SignedChunkRange.FirstBlockHeight())

			processingState := h.factory.CreateProcessingBlocksState(message)
			next := processingState.processState(ctx)

			require.IsType(t, &collectingAvailabilityResponsesState{}, next, "next state after commit error should be collecting availability responses")

			h.verifyMocks(t)
		})
	})
}

func TestStateProcessingBlocksDescending_CommitBlockFailureReturnsToCollectingAvailabilityResponses(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			h := newBlockSyncHarness(harness.Logger).
				withDescendingEnabled(true)
			harness.AllowErrorsMatching("failed to commit block received via sync")

			message := builders.BlockSyncResponseInput().
				WithBlocksOrder(gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING).
				WithFirstBlockHeight(11).
				WithLastBlockHeight(1).
				WithLastCommittedBlockHeight(11).
				Build().Message

			h.storage.When("ValidateChainTip", mock.Any, mock.Any).Return(nil, nil).Times(1)
			h.expectBlockValidationQueriesFromStorage(11)
			h.expectBlockCommitsToStorageAndFailLastCommit(11, message.SignedChunkRange.FirstBlockHeight())

			processingState := h.factory.CreateProcessingBlocksState(message)
			next := processingState.processState(ctx)

			require.IsType(t, &collectingAvailabilityResponsesState{}, next, "next state after commit error should be collecting availability responses")

			h.verifyMocks(t)
		})
	})
}

func TestStateProcessingBlocksAscending_TerminatesOnContextTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	with.Logging(t, func(harness *with.LoggingHarness) {
		h := newBlockSyncHarness(harness.Logger).
			withDescendingEnabled(false)

		message := builders.BlockSyncResponseInput().
			WithBlocksOrder(gossipmessages.SYNC_BLOCKS_ORDER_ASCENDING).
			WithFirstBlockHeight(10).
			WithLastBlockHeight(20).
			WithLastCommittedBlockHeight(20).
			Build().Message

		block := builders.BlockPair().
			WithHeight(primitives.BlockHeight(9)).
			Build()
		syncState := SyncState{InOrderBlock: block, TopBlock: block, LastSyncedBlock: block}
		h.storage.When("GetSyncState").Return(syncState).Times(1)
		h.storage.When("UpdateConsensusAlgosAboutLastCommittedBlockInLocalPersistence", mock.Any)

		cancel()
		state := h.factory.CreateProcessingBlocksState(message)
		nextState := state.processState(ctx)
		time.Sleep(5 * time.Second)

		require.Nil(t, nextState, "next state should be nil on context termination")
	})
}

func TestStateProcessingBlocksDescending_TerminatesOnContextTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	with.Logging(t, func(harness *with.LoggingHarness) {
		h := newBlockSyncHarness(harness.Logger).
			withDescendingEnabled(true)
		message := builders.BlockSyncResponseInput().
			WithBlocksOrder(gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING).
			WithFirstBlockHeight(20).
			WithLastBlockHeight(10).
			WithLastCommittedBlockHeight(20).
			Build().Message

		block := builders.BlockPair().
			WithHeight(primitives.BlockHeight(9)).
			Build()
		syncState := SyncState{InOrderBlock: block, TopBlock: block, LastSyncedBlock: block}
		h.storage.When("GetSyncState").Return(syncState).Times(1)
		h.storage.When("GetBlock", mock.Any).Return(nil)
		h.storage.When("UpdateConsensusAlgosAboutLastCommittedBlockInLocalPersistence", mock.Any)
		h.storage.When("ValidateChainTip", mock.Any, mock.Any).Return(nil, nil).Times(1)

		cancel()
		state := h.factory.CreateProcessingBlocksState(message)
		nextState := state.processState(ctx)
		time.Sleep(5 * time.Second)

		require.Nil(t, nextState, "next state should be nil on context termination")
	})
}
