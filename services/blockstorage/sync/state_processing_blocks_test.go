package sync

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProcessingBlocksCommitsAccordingly(t *testing.T) {
	h := newBlockSyncHarness()

	h.storage.When("ValidateBlockForCommit", mock.Any).Return(nil, nil).Times(11)
	h.storage.When("CommitBlock", mock.Any).Return(nil, nil).Times(11)

	message := builders.BlockSyncResponseInput().
		WithFirstBlockHeight(10).
		WithLastBlockHeight(20).
		WithLastCommittedBlockHeight(20).
		Build().Message

	processingState := h.sf.CreateProcessingBlocksState(message)
	processingState.processState(h.ctx)

	h.verifyMocks(t)
}

func TestProcessingBlocksMovesToCARAfterCommit(t *testing.T) {
	h := newBlockSyncHarness()

	h.storage.When("ValidateBlockForCommit", mock.Any).Return(nil, nil).Times(11)
	h.storage.When("CommitBlock", mock.Any).Return(nil, nil).Times(11)

	message := builders.BlockSyncResponseInput().
		WithFirstBlockHeight(10).
		WithLastBlockHeight(20).
		WithLastCommittedBlockHeight(20).
		Build().Message

	processingState := h.sf.CreateProcessingBlocksState(message)
	next := processingState.processState(h.ctx)

	require.IsType(t, &collectingAvailabilityResponsesState{}, next, "next state after commit should be collecting availability responses")
}

func TestProcessingWithNoBlocksReturnsToIdle(t *testing.T) {
	h := newBlockSyncHarness()

	processingState := h.sf.CreateProcessingBlocksState(nil)
	next := processingState.processState(h.ctx)

	require.IsType(t, &idleState{}, next, "commit initialized invalid should move to idle")
}
