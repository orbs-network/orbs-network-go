package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestService_SlowBlockCreationDoesNotHurtBlockSync(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeanHelixServiceHarness().start(t, ctx)

		blockRequested := make(chan bool)
		h.expectConsensusContextRequestOrderingCommittee(0) // we're index 0
		h.consensusContextRespondWithVerySlowBlockCreation(blockRequested)

		h.consensus.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
			Mode:                   handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY,
			BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
			BlockPair:              nil,
			PrevCommittedBlockPair: nil,
		})

		<-blockRequested
		h.expectConsensusContextRequestOrderingCommittee(1) // we're index 0

		b := builders.BlockPair().WithHeight(1).WithEmptyLeanHelixBlockProof().Build()
		h.consensus.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
			Mode:                   handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY,
			BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
			BlockPair:              b,
			PrevCommittedBlockPair: nil,
		})

		require.NoError(t, test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT, h.consensusContext))
	})
}

func (h *harness) consensusContextRespondWithVerySlowBlockCreation(blockRequested chan bool) {
	h.consensusContext.When("RequestNewTransactionsBlock", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.RequestNewTransactionsBlockInput) (*services.RequestNewTransactionsBlockOutput, error) {
		blockRequested <- true
		<-ctx.Done()
		return nil, errors.New("canceled")
	})
}
