package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/internodesync"
	"github.com/orbs-network/orbs-network-go/services/gossip"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestServiceBlockSync_TransactionPool(t *testing.T) {

	blockCount := primitives.BlockHeight(10)
	txBuilders := make([]*builders.TransactionBuilder, blockCount)
	for i := 0; i < int(blockCount); i++ {
		txBuilders[i] = builders.TransferTransaction().WithAmountAndTargetAddress(uint64(i+1)*10, builders.ClientAddressForEd25519SignerForTests(6))
	}

	newHarness(t).
		AllowingErrors(
			"leader failed to save block to storage",                 // (block already in storage, skipping) TODO(v1) investigate and explain, or fix and remove expected error
			"all consensus \\d* algos refused to validate the block", //TODO(v1) investigate and explain, or fix and remove expected error
		).StartWithRestart(func(ctx context.Context, network NetworkHarness, restartPreservingBlocks func() NetworkHarness) {

		var topBlockHeight primitives.BlockHeight
		for _, builder := range txBuilders {
			resp, _ := network.SendTransaction(ctx, builder.Builder(), 0)
			require.EqualValues(t, protocol.TRANSACTION_STATUS_COMMITTED, resp.TransactionStatus(), "expected transaction to be committed")
			topBlockHeight = resp.RequestResult().BlockHeight()
		}

		network = restartPreservingBlocks()

		_ = network.GetTransactionPoolBlockHeightTracker(0).WaitForBlock(ctx, topBlockHeight)
		_ = network.GetTransactionPoolBlockHeightTracker(1).WaitForBlock(ctx, topBlockHeight)

		// Resend an already committed transaction to Leader
		leaderTxResponse, _ := network.SendTransaction(ctx, txBuilders[0].Builder(), 0)
		nonLeaderTxResponse, _ := network.SendTransaction(ctx, txBuilders[0].Builder(), 1)

		require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED.String(), leaderTxResponse.TransactionStatus().String(),
			"expected a tx that is committed prior to restart and sent again to leader to be rejected")
		require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED.String(), nonLeaderTxResponse.TransactionStatus().String(),
			"expected a tx that is committed prior to restart and sent again to non leader to be rejected")
	})
}

func TestServiceBlockSync_StateStorage(t *testing.T) {

	const transferAmount = 10
	const transfers = 10
	const totalAmount = transfers * transferAmount

	newHarness(t).
		WithLogFilters(log.ExcludeField(gossip.LogTag),
			log.IgnoreMessagesMatching("Metric recorded"),
			log.ExcludeField(internodesync.LogTag)).
		StartWithRestart(func(ctx context.Context, network NetworkHarness, restartPreservingBlocks func() NetworkHarness) {

			var txHashes []primitives.Sha256
			// generate some blocks with state
			contract := network.DeployBenchmarkTokenContract(ctx, 0)
			for i := 0; i < transfers; i++ {
				_, txHash := contract.Transfer(ctx, 0, transferAmount, 0, 1)
				txHashes = append(txHashes, txHash)
			}

			for _, txHash := range txHashes {
				network.BlockPersistence(0).WaitForTransaction(ctx, txHash)
			}

			network = restartPreservingBlocks()

			// wait for all tx to reach state storage:
			for _, txHash := range txHashes {
				network.WaitForTransactionInState(ctx, txHash)
			}

			// verify state in both nodes
			contract = network.DeployBenchmarkTokenContract(ctx, 0)
			balanceNode0 := contract.GetBalance(ctx, 0, 1)
			balanceNode1 := contract.GetBalance(ctx, 1, 1)

			require.EqualValues(t, totalAmount, balanceNode0, "expected transfers to reflect in leader state")
			require.EqualValues(t, totalAmount, balanceNode1, "expected transfers to reflect in non leader state")
		})

}
