// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLeanHelix_RecoversFromDiskError(t *testing.T) {
	newHarness().
		// TODO - reduce sync timeout to speed up test
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		AllowingErrors(
			"failed to commit block received via sync",
			"cannot get elected validators from system contract", // LH tries to read state from a block height that has not been properly persisted and therefore, fails
		).
		Start(t, func(t testing.TB, ctx context.Context, network *Network) {
			r := rand.NewControlledRand(t)
			tamperedNode := r.Intn(len(network.Nodes))
			t.Log("Tampering with node", tamperedNode)

			if err := network.BlockPersistence(tamperedNode).GetBlockTracker().WaitForBlock(ctx, 3); err != nil {
				t.Errorf("waiting for block on node %d failed: %s", tamperedNode, err)
			}

			blocksWhichFailedToPersist := make(chan *protocol.BlockPairContainer, 100) // large buffer
			network.BlockPersistence(tamperedNode).TamperWithBlockWrites(blocksWhichFailedToPersist)

			// wait for two block write failures to occur
			tamperedBlock1 := waitForTamperedBlock(ctx, t, blocksWhichFailedToPersist) // consensus/sync
			tamperedBlock2 := waitForTamperedBlock(ctx, t, blocksWhichFailedToPersist) // sync

			require.Equal(t, heightOf(tamperedBlock1), heightOf(tamperedBlock2), "expected the same height to be attempted after write failure")

			network.BlockPersistence(tamperedNode).ResetTampering()

			// TODO - instead of Eventually, increase the grace period and distance on block tracker
			var err error
			require.Truef(t, test.Eventually(30*time.Second, func() bool {
				err = network.BlockPersistence(tamperedNode).GetBlockTracker().WaitForBlock(ctx, heightOf(tamperedBlock2)+3)
				return err == nil
			}), "waiting for block on node %d failed: %s", tamperedNode, err)
		})
}

func waitForTamperedBlock(ctx context.Context, t testing.TB, failedBlocks chan *protocol.BlockPairContainer) *protocol.BlockPairContainer {
	select {
	case tamperedBlock := <-failedBlocks:
		return tamperedBlock
	case <-ctx.Done():
		t.Fatal("Timed out waiting for block to fail writing")
		return nil
	}
}

func heightOf(blockPairContainer *protocol.BlockPairContainer) primitives.BlockHeight {
	return blockPairContainer.ResultsBlock.Header.BlockHeight()
}
