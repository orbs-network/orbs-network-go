// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/stretchr/testify/require"
	"testing"
)

// There is no need to test more than one consensus algo, because the SUT here is the node-sync mechanism, not the consensus algo
// (could have mocked the whole thing)
// Either test with Benchmark Consensus which is makes it easier to generate fake proofs, or use real recorded Lean Helix blocks
func TestInterNodeBlockSync_WithBenchmarkConsensusBlocks(t *testing.T) {

	newHarness().
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS).
		//WithLogFilters(log.ExcludeEntryPoint("BenchmarkConsensus.Tick")).
		AllowingErrors(
			"leader failed to save block to storage",                 // (block already in storage, skipping) TODO(v1) investigate and explain, or fix and remove expected error
			"all consensus \\d* algos refused to validate the block", //TODO(v1) investigate and explain, or fix and remove expected error
		).
		WithSetup(func(ctx context.Context, network *NetworkHarness) {
			var prevBlock *protocol.BlockPairContainer
			for i := 1; i <= 10; i++ {
				//blockPair := builders.LeanHelixBlockPair().
				blockPair := builders.BenchmarkConsensusBlockPair().
					WithHeight(primitives.BlockHeight(i)).
					WithTransactions(2).
					WithPrevBlock(prevBlock).
					Build()
				network.BlockPersistence(0).WriteNextBlock(blockPair)
				prevBlock = blockPair
			}

			numBlocks, err := network.BlockPersistence(1).GetLastBlockHeight()
			require.NoError(t, err)
			require.Zero(t, numBlocks)
		}).Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
		if err := network.BlockPersistence(0).GetBlockTracker().WaitForBlock(ctx, 10); err != nil {
			t.Errorf("waiting for block on node 0 failed: %s", err)
		}

		if err := network.BlockPersistence(1).GetBlockTracker().WaitForBlock(ctx, 5); err != nil {
			t.Errorf("waiting for block on node 1 failed: %s", err)
		}

		// Wait until full sync
		if err := network.BlockPersistence(1).GetBlockTracker().WaitForBlock(ctx, 10); err != nil {
			t.Errorf("waiting for block on node 1 failed: %s", err)
		}

		// Wait again to get new blocks created after the sync
		if err := network.BlockPersistence(1).GetBlockTracker().WaitForBlock(ctx, 15); err != nil {
			t.Errorf("waiting for block on node 1 failed: %s", err)
		}
	})
}
