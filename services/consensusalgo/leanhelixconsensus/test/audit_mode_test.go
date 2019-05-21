package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestHandleBlockConsensus_ExecutesBlocksYoungerThanThreshold_AndModeIsVerify(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeanHelixServiceHarness(5*time.Minute).start(t, ctx)

		block := builders.BlockPair().WithHeight(1).WithEmptyLeanHelixBlockProof().Build()

		rnrbi := &services.RequestNewResultsBlockInput{
			TransactionsBlock:  block.TransactionsBlock,
			CurrentBlockHeight: block.TransactionsBlock.Header.BlockHeight(),
			PrevBlockHash:      block.TransactionsBlock.Header.PrevBlockHashPtr(),
		}

		h.consensusContext.When("RequestNewResultsBlock", mock.Any, rnrbi).
			Return(&services.RequestNewResultsBlockOutput{
				ResultsBlock: builders.BlockPair().Build().ResultsBlock,
			}, nil).Times(1)

		h.consensus.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
			Mode:                   handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_ONLY,
			BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
			BlockPair:              block,
			PrevCommittedBlockPair: nil,
		})

		_, err := h.consensusContext.Verify()
		require.NoError(t, err, "Consensus Context not invoked as expected")
	})
}

func TestHandleBlockConsensus_DoesNotExecuteBlocksOlderThanThreshold_AndModeIsVerify(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeanHelixServiceHarness(0).start(t, ctx)

		block := builders.BlockPair().WithTimestampAheadBy(-1 * time.Nanosecond).WithHeight(1).WithEmptyLeanHelixBlockProof().Build()

		h.consensusContext.Never("RequestNewResultsBlock", mock.Any, mock.Any)

		h.consensus.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
			Mode:                   handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_ONLY,
			BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
			BlockPair:              block,
			PrevCommittedBlockPair: nil,
		})

		_, err := h.consensusContext.Verify()
		require.NoError(t, err, "Consensus Context not invoked as expected")
	})
}
