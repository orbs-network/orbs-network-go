package externalsync

import (
	"context"
	"errors"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProcessingBlocksCommitsAccordinglyAndMovesToCAR(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness()
		h.storage.When("ValidateBlockForCommit", mock.Any, mock.Any).Return(nil, nil).Times(11)

		outCommit := &services.CommitBlockOutput{}
		h.storage.When("CommitBlock", mock.Any, mock.Any).Return(outCommit, nil).Times(11)

		message := builders.BlockSyncResponseInput().
			WithFirstBlockHeight(10).
			WithLastBlockHeight(20).
			WithLastCommittedBlockHeight(20).
			Build().Message

		processingState := h.factory.CreateProcessingBlocksState(message)
		next := processingState.processState(ctx)

		require.IsType(t, &collectingAvailabilityResponsesState{}, next, "next state after commit should be collecting availability responses")

		h.verifyMocks(t)
	})
}

func TestProcessingWithNoBlocksReturnsToIdle(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness()

		processingState := h.factory.CreateProcessingBlocksState(nil)
		next := processingState.processState(ctx)

		require.IsType(t, &idleState{}, next, "commit initialized invalid should move to idle")
	})
}

func TestProcessingValidationFailureReturnsToCAR(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness()

		message := builders.BlockSyncResponseInput().
			WithFirstBlockHeight(10).
			WithLastBlockHeight(20).
			WithLastCommittedBlockHeight(20).
			Build().Message

		h.storage.When("ValidateBlockForCommit", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error) {
			if input.BlockPair.ResultsBlock.Header.BlockHeight().Equal(message.SignedChunkRange.FirstBlockHeight() + 5) {
				return nil, errors.New("failed to validate block #6")
			}
			return nil, nil
		}).Times(6)
		h.storage.When("CommitBlock", mock.Any, mock.Any).Return(nil, nil).Times(5)

		processingState := h.factory.CreateProcessingBlocksState(message)
		next := processingState.processState(ctx)

		require.IsType(t, &collectingAvailabilityResponsesState{}, next, "next state after validation error should be collecting availability responses")

		h.verifyMocks(t)
	})
}

func TestProcessingCommitFailureReturnsToCAR(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness()

		message := builders.BlockSyncResponseInput().
			WithFirstBlockHeight(10).
			WithLastBlockHeight(20).
			WithLastCommittedBlockHeight(20).
			Build().Message

		h.storage.When("ValidateBlockForCommit", mock.Any, mock.Any).Return(nil, nil).Times(6)
		h.storage.When("CommitBlock", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
			if input.BlockPair.ResultsBlock.Header.BlockHeight().Equal(message.SignedChunkRange.FirstBlockHeight() + 5) {
				return nil, errors.New("failed to commit block #6")
			}
			return nil, nil
		}).Times(6)

		processingState := h.factory.CreateProcessingBlocksState(message)
		next := processingState.processState(ctx)

		require.IsType(t, &collectingAvailabilityResponsesState{}, next, "next state after commit error should be collecting availability responses")

		h.verifyMocks(t)
	})
}

func TestProcessingContextTerminationFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h := newBlockSyncHarness()
	cancel()

	message := builders.BlockSyncResponseInput().
		WithFirstBlockHeight(10).
		WithLastBlockHeight(20).
		WithLastCommittedBlockHeight(20).
		Build().Message

	processingState := h.factory.CreateProcessingBlocksState(message)
	next := processingState.processState(ctx)

	require.Nil(t, next, "next state should be nil on context termination")
}

func TestProcessingNOP(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness()
		processing := h.factory.CreateProcessingBlocksState(nil)

		// these tests are for sanity, they should not do anything
		processing.blockCommitted(ctx)
		processing.gotBlocks(ctx, nil)
		processing.gotAvailabilityResponse(ctx, nil)
	})
}
