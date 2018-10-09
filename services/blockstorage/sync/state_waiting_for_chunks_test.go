package sync

import (
	"errors"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestWaitingMovedToIdleOnTransportError(t *testing.T) {
	h := newBlockSyncHarness()

	h.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(0)).Times(1)
	h.gossip.When("SendBlockSyncRequest", mock.Any).Return(nil, errors.New("gossip failure")).Times(1)

	waitingState := h.sf.CreateWaitingForChunksState(h.config.NodePublicKey())
	nextState := waitingState.processState(h.ctx)

	_, isIdle := nextState.(*idleState)

	require.True(t, isIdle, "expecting back to idle on transport error")

	h.verifyMocks(t)
}

func TestWaitingMovesToIdleOnTimeout(t *testing.T) {
	h := newBlockSyncHarness()

	h.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(0)).Times(1)
	h.gossip.When("SendBlockSyncRequest", mock.Any).Return(nil, nil).Times(1)

	waitingState := h.sf.CreateWaitingForChunksState(h.config.NodePublicKey())
	nextState := waitingState.processState(h.ctx)

	_, isIdle := nextState.(*idleState)

	require.True(t, isIdle, "expecting back to idle on transport error")

	h.verifyMocks(t)
}

func TestWaitingAcceptsNewBlockAndMovesToProcessing(t *testing.T) {
	blocksMessage := builders.BlockSyncResponseInput().Build().Message
	h := newBlockSyncHarness().withNodeKey(blocksMessage.Sender.SenderPublicKey()).withWaitForChunksTimeout(10 * time.Millisecond)

	h.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)
	h.gossip.When("SendBlockSyncRequest", mock.Any).Return(nil, nil).Times(1)

	waitingState := h.sf.CreateWaitingForChunksState(h.config.NodePublicKey())
	var nextState syncState
	latch := make(chan struct{})
	go func() {
		nextState = waitingState.processState(h.ctx)
		latch <- struct{}{}
	}()
	waitingState.gotBlocks(blocksMessage)
	<-latch

	require.IsType(t, &processingBlocksState{}, nextState, "expecting to be at processing state after blocks arrived")

	h.verifyMocks(t)
}

func TestWaitingTerminatesOnContextTermination(t *testing.T) {
	h := newBlockSyncHarness().withWaitForChunksTimeout(3 * time.Millisecond)
	h.cancel()

	h.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)
	h.gossip.When("SendBlockSyncRequest", mock.Any).Return(nil, nil).Times(1)

	waitingState := h.sf.CreateWaitingForChunksState(h.config.NodePublicKey())
	nextState := waitingState.processState(h.ctx)

	require.Nil(t, nextState, "context terminated, expected nil state")
}

func TestWaitingMovesToIdleOnIncorrectMessageSource(t *testing.T) {
	messageSourceKey := keys.Ed25519KeyPairForTests(1).PublicKey()
	blocksMessage := builders.BlockSyncResponseInput().WithSenderPublicKey(messageSourceKey).Build().Message
	stateSourceKey := keys.Ed25519KeyPairForTests(8).PublicKey()
	h := newBlockSyncHarness().withNodeKey(stateSourceKey).withWaitForChunksTimeout(10 * time.Millisecond)

	h.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)
	h.gossip.When("SendBlockSyncRequest", mock.Any).Return(nil, nil).Times(1)

	waitingState := h.sf.CreateWaitingForChunksState(h.config.NodePublicKey())
	var nextState syncState
	latch := make(chan struct{})
	go func() {
		nextState = waitingState.processState(h.ctx)
		latch <- struct{}{}
	}()
	waitingState.gotBlocks(blocksMessage)
	<-latch

	require.IsType(t, &idleState{}, nextState, "expecting to abort sync and go back to idle (ignore blocks)")

	h.verifyMocks(t)
}

func TestWaitingNOP(t *testing.T) {
	h := newBlockSyncHarness().withWaitForChunksTimeout(3 * time.Millisecond)
	waitingState := h.sf.CreateWaitingForChunksState(h.config.NodePublicKey())

	// this is sanity, these calls should do nothing
	waitingState.gotAvailabilityResponse(nil)
	waitingState.blockCommitted(primitives.BlockHeight(0))
}
