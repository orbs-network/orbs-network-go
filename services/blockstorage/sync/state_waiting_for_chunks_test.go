package sync

import (
	"errors"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
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
