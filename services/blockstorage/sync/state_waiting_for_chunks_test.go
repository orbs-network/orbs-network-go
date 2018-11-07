package sync

import (
	"errors"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestWaitingMovedToIdleOnTransportError(t *testing.T) {
	h := newBlockSyncHarness()

	h.expectLastCommittedBlockHeight(primitives.BlockHeight(0))
	h.gossip.When("SendBlockSyncRequest", mock.Any, mock.Any).Return(nil, errors.New("gossip failure")).Times(1)

	waitingState := h.factory.CreateWaitingForChunksState(h.config.NodePublicKey())
	nextState := waitingState.processState(h.ctx)

	require.IsType(t, &idleState{}, nextState, "expecting back to idle on transport error")

	h.verifyMocks(t)
}

func TestWaitingMovesToIdleOnTimeout(t *testing.T) {
	h := newBlockSyncHarness()

	h.expectLastCommittedBlockHeight(primitives.BlockHeight(0))
	h.gossip.When("SendBlockSyncRequest", mock.Any, mock.Any).Return(nil, nil).Times(1)

	waitingState := h.factory.CreateWaitingForChunksState(h.config.NodePublicKey())
	nextState := waitingState.processState(h.ctx)

	require.IsType(t, &idleState{}, nextState, "expecting back to idle on timeout")

	h.verifyMocks(t)
}

func TestWaitingAcceptsNewBlockAndMovesToProcessing(t *testing.T) {
	manualWaitForChunksTimer := synchronization.NewTimerWithManualTick()
	blocksMessage := builders.BlockSyncResponseInput().Build().Message
	h := newBlockSyncHarnessWithManualWaitForChunksTimeoutTimer(manualWaitForChunksTimer).withNodeKey(blocksMessage.Sender.SenderPublicKey())

	h.expectLastCommittedBlockHeight(primitives.BlockHeight(10))
	h.gossip.When("SendBlockSyncRequest", mock.Any, mock.Any).Return(nil, nil).Times(1)

	waitingState := h.factory.CreateWaitingForChunksState(h.config.NodePublicKey())
	nextState := h.processStateAndWaitUntilFinished(waitingState, func() {
		waitingState.gotBlocks(h.ctx, blocksMessage)
		manualWaitForChunksTimer.ManualTick()
	})

	require.IsType(t, &processingBlocksState{}, nextState, "expecting to be at processing state after blocks arrived")
	pbs := nextState.(*processingBlocksState)
	require.NotNil(t, pbs.blocks, "blocks payload initialized in processing stage")
	require.Equal(t, blocksMessage.Sender, pbs.blocks.Sender, "expected sender in source message to be the same in the state")
	require.Equal(t, len(blocksMessage.BlockPairs), len(pbs.blocks.BlockPairs), "expected same number of blocks in message->state")
	require.Equal(t, blocksMessage.SignedChunkRange, pbs.blocks.SignedChunkRange, "expected signed range to be the same in message -> state")

	h.verifyMocks(t)
}

func TestWaitingTerminatesOnContextTermination(t *testing.T) {
	h := newBlockSyncHarness()
	h.cancel()

	h.expectLastCommittedBlockHeight(primitives.BlockHeight(10))
	h.gossip.When("SendBlockSyncRequest", mock.Any, mock.Any).Return(nil, nil).Times(1)

	waitingState := h.factory.CreateWaitingForChunksState(h.config.NodePublicKey())
	nextState := waitingState.processState(h.ctx)

	require.Nil(t, nextState, "context terminated, expected nil state")
}

func TestWaitingMovesToIdleOnIncorrectMessageSource(t *testing.T) {
	messageSourceKey := keys.Ed25519KeyPairForTests(1).PublicKey()
	blocksMessage := builders.BlockSyncResponseInput().WithSenderPublicKey(messageSourceKey).Build().Message
	stateSourceKey := keys.Ed25519KeyPairForTests(8).PublicKey()
	h := newBlockSyncHarness().withNodeKey(stateSourceKey)

	h.expectLastCommittedBlockHeight(primitives.BlockHeight(10))
	h.gossip.When("SendBlockSyncRequest", mock.Any, mock.Any).Return(nil, nil).Times(1)

	waitingState := h.factory.CreateWaitingForChunksState(h.config.NodePublicKey())
	nextState := h.processStateAndWaitUntilFinished(waitingState, func() {
		waitingState.gotBlocks(h.ctx, blocksMessage)
	})

	require.IsType(t, &idleState{}, nextState, "expecting to abort sync and go back to idle (ignore blocks)")

	h.verifyMocks(t)
}

func TestWaitingDoesNotBlockOnBlocksNotificationWhenChannelIsNotReady(t *testing.T) {
	h := newBlockSyncHarness()
	h = h.withCtxTimeout(h.config.collectChunks / 2)
	waitingState := h.factory.CreateWaitingForChunksState(h.config.NodePublicKey())
	messageSourceKey := keys.Ed25519KeyPairForTests(1).PublicKey()
	blocksMessage := builders.BlockSyncResponseInput().WithSenderPublicKey(messageSourceKey).Build().Message
	waitingState.gotBlocks(h.ctx, blocksMessage) // we did not call process, so channel is not ready, test fails if this blocks
	h.cancel()
}

func TestWaitingNOP(t *testing.T) {
	h := newBlockSyncHarness()
	waitingState := h.factory.CreateWaitingForChunksState(h.config.NodePublicKey())

	// this is sanity, these calls should do nothing
	waitingState.gotAvailabilityResponse(h.ctx, nil)
	waitingState.blockCommitted(h.ctx)
}
