// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLeanHelix_RecoversFromDiskError(t *testing.T) {
	t.Skipf("Failing test - will pass when LH is fixed. see issue https://github.com/orbs-network/orbs-network-go/issues/1253")
	newHarness().
		// TODO - reduce sync timeout to speed up test
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		AllowingErrors(
			"failed to commit block received via sync",
			"cannot get elected validators from system contract", // LH tries to read state from a block height that has not been properly persisted and therefore, fails
		).
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			r := rand.NewControlledRand(t)
			tamperedNode := r.Intn(len(network.Nodes))
			t.Log("Tampering with node", tamperedNode)

			if err := network.BlockPersistence(tamperedNode).GetBlockTracker().WaitForBlock(ctx, 5); err != nil {
				t.Errorf("waiting for block on node %d failed: %s", tamperedNode, err)
			}

			failedBlocks := make(chan *protocol.BlockPairContainer, 100)
			network.BlockPersistence(tamperedNode).FailBlockWrites(failedBlocks)

			// wait for two block write failures to occur
			tamperedBlock1 := <-failedBlocks // consensus
			tamperedBlock2 := <-failedBlocks // sync

			require.Equal(t, tamperedBlock1.ResultsBlock.Header.BlockHeight(), tamperedBlock2.ResultsBlock.Header.BlockHeight(), "expected the same height to be attempted after write failure")

			// how far consensus advanced meanwhile
			healthyNode := (tamperedNode + 1) % len(network.Nodes)
			topHeight, err := network.BlockPersistence(healthyNode).GetLastBlockHeight()
			require.NoError(t, err)
			require.Condition(t, func() (success bool) {
				return topHeight > tamperedBlock2.ResultsBlock.Header.BlockHeight()
			}, "expected tampered node to fall behind the pack")

			network.BlockPersistence(tamperedNode).ResetTampering()

			// TODO - instead of Eventually, increase the grace period and distance on block tracker
			require.True(t, test.Eventually(30*time.Second, func() bool {
				err = network.BlockPersistence(tamperedNode).GetBlockTracker().WaitForBlock(ctx, topHeight+5)
				return err == nil
			}), fmt.Sprintf("waiting for block on node %d failed: %s", tamperedNode, err))
		})
}
