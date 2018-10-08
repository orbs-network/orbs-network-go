package sync

import (
	"errors"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test/builders"
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
	h := newBlockSyncHarness().WithNodeKey(blocksMessage.Sender.SenderPublicKey()).WithWaitForChunksTimeout(10 * time.Millisecond)

	h.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)
	h.gossip.When("SendBlockSyncRequest", mock.Any).Return(nil, nil).Times(1)

	waitingState := h.sf.CreateWaitingForChunksState(h.config.NodePublicKey())
	var nextState syncState
	latch := make(chan struct{})
	go func() {
		nextState = waitingState.processState(h.ctx)
		latch <- struct{}{}
	}()
	waitingState.gotBlocks(blocksMessage.Sender.SenderPublicKey(), blocksMessage)
	<-latch

	require.IsType(t, &processingBlocksState{}, nextState, "expecting to be at processing state after blocks arrived")

	h.verifyMocks(t)
}

func TestWaitingTerminatesOnContextTermination(t *testing.T) {
	h := newBlockSyncHarness().WithWaitForChunksTimeout(3 * time.Millisecond)
	h.Cancel()

	h.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)
	h.gossip.When("SendBlockSyncRequest", mock.Any).Return(nil, nil).Times(1)

	waitingState := h.sf.CreateWaitingForChunksState(h.config.NodePublicKey())
	nextState := waitingState.processState(h.ctx)

	require.Nil(t, nextState, "context terminated, expected nil state")
}
