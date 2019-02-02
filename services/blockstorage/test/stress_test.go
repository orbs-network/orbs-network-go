package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// this test tries to emulate a potential deadlock between block sync and consensus algo.
// if consensus algo does not respond to HandleBlockConsensus() while committing new blocks a deadlock may happen:
// BlockSync calls ConsensusAlgo.HandleBlockConsensus() when sync wakes up.
// ConsensusAlgo calls BlockStorage.CommitBlock() when a new block is closed.
func TestSyncPetitioner_Stress_SingleThreadedConsensusAlgoDoesNotDeadlock(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness().withSyncNoCommitTimeout(time.Nanosecond).start(ctx)
		harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Return(nil, nil).AtLeast(0)

		updateConsensusAlgoHeight := make(chan struct{})

		targetBlockHeight := primitives.BlockHeight(100)
		startFakeSingleThreadedConsensusAlgo(t, ctx, harness, targetBlockHeight, updateConsensusAlgoHeight)

		var topReportedHeight primitives.BlockHeight
		harness.consensus.Reset().When("HandleBlockConsensus", mock.Any, mock.Any).Call(func(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {
			if input.BlockPair != nil {
				updateConsensusAlgoHeight <- struct{}{}
				topReportedHeight = input.BlockPair.ResultsBlock.Header.BlockHeight()
			}

			return nil, nil
		}).AtLeast(0)

		require.Truef(t, test.Eventually(10*time.Second, func() bool {
			return topReportedHeight == targetBlockHeight
		}), "expected blocks to be produced without deadlock, but only %d were closed", topReportedHeight)
	})
}

// emulates an inconsiderate ConsensusAlgo that blocks HandleBlockConsensus() calls while committing blocks, and closes Blocks eagerly.
func startFakeSingleThreadedConsensusAlgo(t *testing.T, ctx context.Context, harness *harness, targetBlockHeight primitives.BlockHeight, updateConsensusAlgoHeight <-chan struct{}) {
	var h primitives.BlockHeight
	go func() {
		for {
			select {
			case <-ctx.Done():
			case <-updateConsensusAlgoHeight:
			default:
				if h < targetBlockHeight {
					time.Sleep(time.Nanosecond)
					h++
					_, err := harness.commitBlock(ctx, builders.BlockPair().WithHeight(h).WithTimestampNow().Build())
					require.NoError(t, err)
				}
			}
		}
	}()
}
