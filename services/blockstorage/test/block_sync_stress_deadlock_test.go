// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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
	test.WithConcurrencyHarness(t, func(ctx context.Context, parent *test.ConcurrencyHarness) {
		harness := newBlockStorageHarness(parent).withSyncNoCommitTimeout(time.Nanosecond)
		harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Return(nil, nil).AtLeast(0)

		updateConsensusAlgoHeight := make(chan struct{})

		targetBlockHeight := primitives.BlockHeight(100)

		committedBlockHeights := make(chan primitives.BlockHeight, 100)
		harness.consensus.When("HandleBlockConsensus", mock.Any, mock.Any).Call(func(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {
			if input.BlockPair != nil {
				updateConsensusAlgoHeight <- struct{}{}
				committedBlockHeights <- input.BlockPair.ResultsBlock.Header.BlockHeight()
			}

			return nil, nil
		}).AtLeast(0)

		harness.start(ctx)
		startFakeSingleThreadedConsensusAlgo(t, ctx, harness, targetBlockHeight, updateConsensusAlgoHeight)

		waitUntilReachedBlockHeight(ctx, t, committedBlockHeights, targetBlockHeight)
	})
}

func waitUntilReachedBlockHeight(ctx context.Context, t *testing.T, committedBlockHeights chan primitives.BlockHeight, targetBlockHeight primitives.BlockHeight) {
	var topReportedHeight primitives.BlockHeight
	for {
		select {
		case topReportedHeight = <-committedBlockHeights:
			if topReportedHeight >= targetBlockHeight {
				return
			}
		case <-ctx.Done():
			t.Errorf("expected blocks to be produced without deadlock, but only %d were closed", topReportedHeight)
		}
	}
}

// emulates an inconsiderate ConsensusAlgo that blocks HandleBlockConsensus() calls while committing blocks, and closes Blocks eagerly.
func startFakeSingleThreadedConsensusAlgo(t *testing.T, ctx context.Context, harness *harness, targetBlockHeight primitives.BlockHeight, updateConsensusAlgoHeight <-chan struct{}) {
	var h primitives.BlockHeight
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-updateConsensusAlgoHeight:
			default:
				if h < targetBlockHeight {
					h++
					_, err := harness.commitBlock(ctx, builders.BlockPair().WithHeight(h).WithTimestampNow().Build())
					require.NoError(t, err)
					time.Sleep(time.Nanosecond)
				}
			}
		}
	}()
}
