package internodesync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStateWaitingForChunks_MovesToIdleOnTransportError(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness()

		h.expectLastCommittedBlockHeightQueryFromStorage(0)
		h.expectSendingOfBlockSyncRequestToFail()

		state := h.factory.CreateWaitingForChunksState(h.config.NodeAddress())
		nextState := state.processState(ctx)

		require.IsType(t, &idleState{}, nextState, "expecting back to idle on transport error")
		h.verifyMocks(t)
	})
}

func TestStateWaitingForChunks_MovesToIdleOnTimeout(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness()

		h.expectLastCommittedBlockHeightQueryFromStorage(0)
		h.expectSendingOfBlockSyncRequest()

		state := h.factory.CreateWaitingForChunksState(h.config.NodeAddress())
		nextState := state.processState(ctx)

		require.IsType(t, &idleState{}, nextState, "expecting back to idle on timeout")
		h.verifyMocks(t)
	})
}

func TestStateWaitingForChunks_AcceptsNewBlockAndMovesToProcessingBlocks(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		manualWaitForChunksTimer := synchronization.NewTimerWithManualTick()
		blocksMessage := builders.BlockSyncResponseInput().Build().Message
		h := newBlockSyncHarnessWithManualWaitForChunksTimeoutTimer(func() *synchronization.Timer {
			return manualWaitForChunksTimer
		}).withNodeAddress(blocksMessage.Sender.SenderNodeAddress())

		h.expectLastCommittedBlockHeightQueryFromStorage(10)
		h.expectSendingOfBlockSyncRequest()

		state := h.factory.CreateWaitingForChunksState(h.config.NodeAddress())
		nextState := h.processStateInBackgroundAndWaitUntilFinished(ctx, state, func() {
			h.factory.conduit <- blocksMessage
			manualWaitForChunksTimer.ManualTick() // not required, added for completion (like in state_availability_requests_test)
		})

		require.IsType(t, &processingBlocksState{}, nextState, "expecting to be at processing state after blocks arrived")
		pbs := nextState.(*processingBlocksState)
		require.NotNil(t, pbs.blocks, "blocks payload initialized in processing stage")
		require.Equal(t, blocksMessage.Sender, pbs.blocks.Sender, "expected sender in source message to be the same in the state")
		require.Equal(t, len(blocksMessage.BlockPairs), len(pbs.blocks.BlockPairs), "expected same number of blocks in message->state")
		require.Equal(t, blocksMessage.SignedChunkRange, pbs.blocks.SignedChunkRange, "expected signed range to be the same in message -> state")

		h.verifyMocks(t)
	})
}

func TestStateWaitingForChunks_TerminatesOnContextTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h := newBlockSyncHarness()

	h.expectLastCommittedBlockHeightQueryFromStorage(10)
	h.expectSendingOfBlockSyncRequest()

	cancel()
	state := h.factory.CreateWaitingForChunksState(h.config.NodeAddress())
	nextState := state.processState(ctx)

	require.Nil(t, nextState, "context terminated, expected nil state")
}

func TestStateWaitingForChunks_MovesToIdleOnIncorrectMessageSource(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		messageSourceAddress := keys.EcdsaSecp256K1KeyPairForTests(1).NodeAddress()
		blocksMessage := builders.BlockSyncResponseInput().WithSenderNodeAddress(messageSourceAddress).Build().Message
		stateSourceAddress := keys.EcdsaSecp256K1KeyPairForTests(8).NodeAddress()
		h := newBlockSyncHarness().withNodeAddress(stateSourceAddress)

		h.expectLastCommittedBlockHeightQueryFromStorage(10)
		h.expectSendingOfBlockSyncRequest()

		state := h.factory.CreateWaitingForChunksState(h.config.NodeAddress())
		nextState := h.processStateInBackgroundAndWaitUntilFinished(ctx, state, func() {
			h.factory.conduit <- blocksMessage
		})

		require.IsType(t, &idleState{}, nextState, "expecting to abort sync and go back to idle (ignore blocks)")
		h.verifyMocks(t)
	})
}
