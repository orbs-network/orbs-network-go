package sync

import (
	"errors"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/services"
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

func TestProcessingValidationFailure(t *testing.T) {
	h := newBlockSyncHarness()

	message := builders.BlockSyncResponseInput().
		WithFirstBlockHeight(10).
		WithLastBlockHeight(20).
		WithLastCommittedBlockHeight(20).
		Build().Message

	h.storage.When("ValidateBlockForCommit", mock.Any).Call(func(input *services.ValidateBlockForCommitInput) error {
		if input.BlockPair.ResultsBlock.Header.BlockHeight().Equal(message.SignedChunkRange.FirstBlockHeight() + 5) {
			return errors.New("failed to validate block #6")
		}
		return nil
	}).Times(6)
	h.storage.When("CommitBlock", mock.Any).Return(nil, nil).Times(5)

	processingState := h.sf.CreateProcessingBlocksState(message)
	next := processingState.processState(h.ctx)

	require.IsType(t, &collectingAvailabilityResponsesState{}, next, "next state after validation error should be collecting availability responses")

	h.verifyMocks(t)
}
