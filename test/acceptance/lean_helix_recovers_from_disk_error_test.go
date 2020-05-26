// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLeanHelix_RecoversFromDiskWriteError(t *testing.T) {
	NewHarness().
		WithSetup(func(ctx context.Context, network *Network) {
			// set current reference time to now for node sync verifications
			newRefTime := GenerateNewManagementReferenceTime(0)
			err := network.committeeProvider.AddCommittee(newRefTime, testKeys.NodeAddressesForTests()[1:5])
			require.NoError(t, err)
		}).
		// TODO - reduce sync timeout to speed up test
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		AllowingErrors(
			"failed to commit to persistent storage from temp storage", // Temp: this test intentionally fails block writes
			"failed to commit block received via sync",           // this test intentionally fails block writes
			"cannot get elected validators from system contract", // LH tries to read state from a block height that has not been properly persisted and therefore, fails
		).
		Start(t, func(t testing.TB, ctx context.Context, network *Network) {
			r := rand.NewControlledRand(t)
			tamperedNode := r.Intn(len(network.Nodes))
			t.Log("Tampering with node", tamperedNode)

			const livelinessCheckHeightThreshold = 3

			// wait for the network to start closing blocks
			if err := network.BlockPersistence(tamperedNode).GetBlockTracker().WaitForBlock(ctx, livelinessCheckHeightThreshold); err != nil {
				t.Errorf("waiting for block on node %d failed: %s", tamperedNode, err)
			}

			// tamper with block writes
			blocksWhichFailedToPersist := make(chan *protocol.BlockPairContainer, 100) // large buffer
			network.BlockPersistence(tamperedNode).TamperWithBlockWrites(blocksWhichFailedToPersist)

			lastWrittenHeight, err := network.BlockPersistence(tamperedNode).GetLastBlockHeight()
			require.NoError(t, err)
			// wait for two block write failures to occur
			inspectFailedWriteAttempts := 2
			for i := 0; i < inspectFailedWriteAttempts; i++ {
				unwrittenBlock := waitForUnwrittenBlock(ctx, t, blocksWhichFailedToPersist)
				t.Log("Detected an unwritten block height ", unwrittenBlock.ResultsBlock.Header.BlockHeight())
				// typically all block attempts will be of the next expected height. but we are tolerant to previously written block heights as well since they may be retried due to sync race conditions
				require.True(t, heightOf(unwrittenBlock) <= lastWrittenHeight+1, "any block write attempt is expected to be of (at most) the next unwritten height")
			}

			// un-tamper with block writes
			network.BlockPersistence(tamperedNode).ResetTampering()

			proceedToHeight := lastWrittenHeight + livelinessCheckHeightThreshold
			err = eventuallyReachHeight(ctx, network.BlockPersistence(tamperedNode), proceedToHeight)
			require.NoError(t, err, "waiting for block %d on node %d failed: %s", proceedToHeight, tamperedNode, err)
		})
}

func eventuallyReachHeight(ctx context.Context, blockPersistence adapter.BlockPersistence, targetHeight primitives.BlockHeight) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for timeoutCtx.Err() == nil { // until timeout or context dies
		err := blockPersistence.GetBlockTracker().WaitForBlock(timeoutCtx, targetHeight)
		if err == nil {
			return nil // reached expected height
		}
	}
	return timeoutCtx.Err() // why we stopped waiting
}

func waitForUnwrittenBlock(ctx context.Context, t testing.TB, failedBlocks chan *protocol.BlockPairContainer) *protocol.BlockPairContainer {
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
