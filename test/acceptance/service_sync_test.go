// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/internodesync"
	"github.com/orbs-network/orbs-network-go/services/gossip"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestServiceBlockSync_TransactionPool(t *testing.T) {

	blockCount := primitives.BlockHeight(10)
	txBuilders := make([]*builders.TransactionBuilder, blockCount)
	for i := 0; i < int(blockCount); i++ {
		txBuilders[i] = builders.TransferTransaction().WithAmountAndTargetAddress(uint64(i+1)*10, builders.ClientAddressForEd25519SignerForTests(6))
	}

	blocks := createInitialBlocks(t, txBuilders)

	newHarness().
		WithInitialBlocks(blocks).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS). // this test only runs with BenchmarkConsensus since we only create blocks compatible with that algo
		AllowingErrors(
			"leader failed to save block to storage",                 // (block already in storage, skipping) TODO(v1) investigate and explain, or fix and remove expected error
			"all consensus \\d* algos refused to validate the block", //TODO(v1) investigate and explain, or fix and remove expected error
		).Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {

		topBlockHeight := blocks[len(blocks)-1].ResultsBlock.Header.BlockHeight()

		_ = network.GetTransactionPoolBlockHeightTracker(0).WaitForBlock(ctx, topBlockHeight)
		_ = network.GetTransactionPoolBlockHeightTracker(1).WaitForBlock(ctx, topBlockHeight)

		// this is required because GlobalPreOrder contract relies on state (Approve method), and if state storage is too far behind, GlobalPreOrder will fail on gap
		require.NoError(t, network.stateBlockHeightTrackers[0].WaitForBlock(ctx, topBlockHeight))
		require.NoError(t, network.stateBlockHeightTrackers[1].WaitForBlock(ctx, topBlockHeight))

		// Resend an already committed transaction to Leader
		leaderTxResponse, _ := network.SendTransaction(ctx, txBuilders[0].Builder(), 0)
		nonLeaderTxResponse, _ := network.SendTransaction(ctx, txBuilders[0].Builder(), 1)

		require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED.String(), leaderTxResponse.TransactionStatus().String(),
			"expected a tx that is committed prior to restart and sent again to leader to be rejected")
		require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED.String(), nonLeaderTxResponse.TransactionStatus().String(),
			"expected a tx that is committed prior to restart and sent again to non leader to be rejected")
	})
}

func createInitialBlocks(t testing.TB, txBuilders []*builders.TransactionBuilder) (blocks []*protocol.BlockPairContainer) {
	usingABenchmarkConsensusNetwork(t, func(ctx context.Context, network *NetworkHarness) {
		for _, builder := range txBuilders {
			resp, _ := network.SendTransaction(ctx, builder.Builder(), 0)
			require.EqualValues(t, protocol.TRANSACTION_STATUS_COMMITTED, resp.TransactionStatus(), "expected transaction to be committed")
		}

		var err error
		blocks, err = network.Nodes[0].ExtractBlocks()
		require.NoError(t, err, "failed fetching blocks from persistence")
	})

	return
}

func TestServiceBlockSync_StateStorage(t *testing.T) {

	const transferAmount = 10
	const transfers = 10
	const totalAmount = transfers * transferAmount

	blocks, txHashes := createTransferBlocks(t, transfers, transferAmount)

	newHarness().
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS). // this test only runs with BenchmarkConsensus since we only create blocks compatible with that algo
		WithInitialBlocks(blocks).
		WithLogFilters(log.ExcludeField(gossip.LogTag),
			log.ExcludeField(internodesync.LogTag)).
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {

			// wait for all tx to reach state storage:
			for _, txHash := range txHashes {
				network.WaitForTransactionInState(ctx, txHash)
			}

			// verify state in both nodes
			contract := network.DeployBenchmarkTokenContract(ctx, 0)
			balanceNode0 := contract.GetBalance(ctx, 0, 1)
			balanceNode1 := contract.GetBalance(ctx, 1, 1)

			require.EqualValues(t, totalAmount, balanceNode0, "expected transfers to reflect in leader state")
			require.EqualValues(t, totalAmount, balanceNode1, "expected transfers to reflect in non leader state")
		})

}

func createTransferBlocks(t testing.TB, transfers int, amount uint64) (blocks []*protocol.BlockPairContainer, txHashes []primitives.Sha256) {

	usingABenchmarkConsensusNetwork(t, func(ctx context.Context, network *NetworkHarness) {

		// generate some blocks with state
		contract := network.DeployBenchmarkTokenContract(ctx, 0)
		for i := 0; i < transfers; i++ {
			_, txHash := contract.Transfer(ctx, 0, amount, 0, 1)
			txHashes = append(txHashes, txHash)
		}

		for _, txHash := range txHashes {
			network.BlockPersistence(0).WaitForTransaction(ctx, txHash)
		}

		var err error
		blocks, err = network.Nodes[0].ExtractBlocks()
		require.NoError(t, err, "failed generating blocks for test")
	})

	return
}
