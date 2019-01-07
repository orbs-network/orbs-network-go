package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestServiceBlockSync_TransactionPool(t *testing.T) {

	blockCount := primitives.BlockHeight(10)
	txBuilders := make([]*builders.TransactionBuilder, blockCount)
	for i := 0; i < int(blockCount); i++ {
		txBuilders[i] = builders.TransferTransaction().WithAmountAndTargetAddress(uint64(i+1)*10, builders.ClientAddressForEd25519SignerForTests(6))
	}

	harness.Network(t).
		AllowingErrors(
			"leader failed to save block to storage",                 // (block already in storage, skipping) TODO(v1) investigate and explain, or fix and remove expected error
			"all consensus \\d* algos refused to validate the block", //TODO(v1) investigate and explain, or fix and remove expected error
		).StartWithRestart(func(ctx context.Context, network harness.TestNetworkDriver, restartPreservingBlocks func() harness.TestNetworkDriver) {

		var mostRecentTxHash primitives.Sha256

		for _, builder := range txBuilders {
			_, mostRecentTxHash = network.SendTransaction(ctx, builder.Builder(), 0)
		}
		network.WaitForTransactionInNodeState(ctx, mostRecentTxHash, 0)

		network = restartPreservingBlocks()

		require.True(t, waitForTransactionStatusCommitted(ctx, network, mostRecentTxHash, 0),
			"expected tx to be committed to leader tx pool")
		require.True(t, waitForTransactionStatusCommitted(ctx, network, mostRecentTxHash, 1),
			"expected tx to be committed to non leader tx pool")

		// Resend an already committed transaction to Leader
		leaderTxResponse, _ := network.SendTransaction(ctx, txBuilders[0].Builder(), 0)
		require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, leaderTxResponse.TransactionStatus(),
			"expected a stale tx sent to leader to be rejected")

		// Resend an already committed transaction to Non-Leader
		nonLeaderTxResponse, _ := network.SendTransaction(ctx, txBuilders[0].Builder(), 1)
		require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, nonLeaderTxResponse.TransactionStatus(),
			"expected a stale tx sent to non leader to be rejected")
	})
}

func waitForTransactionStatusCommitted(ctx context.Context, network harness.TestNetworkDriver, txHash primitives.Sha256, nodeIndex int) bool {
	return test.Eventually(5*time.Second, func() bool {
		txStatusOut, err := network.PublicApi(nodeIndex).GetTransactionStatus(ctx, &services.GetTransactionStatusInput{
			ClientRequest: (&client.GetTransactionStatusRequestBuilder{
				TransactionRef: builders.TransactionRef().WithTxHash(txHash).Builder(),
			}).Build(),
		})
		if err != nil {
			return false
		}
		return txStatusOut.ClientResponse.TransactionStatus() == protocol.TRANSACTION_STATUS_COMMITTED
	})
}

func TestServiceBlockSync_StateStorage(t *testing.T) {

	const transferAmount = 10
	const transfers = 10
	const totalAmount = transfers * transferAmount

	harness.Network(t).
		AllowingErrors(
			"leader failed to save block to storage",                 // (block already in storage, skipping) TODO(v1) investigate and explain, or fix and remove expected error
			"all consensus \\d* algos refused to validate the block", //TODO(v1) investigate and explain, or fix and remove expected error
		).
		StartWithRestart(func(ctx context.Context, network harness.TestNetworkDriver, restartPreservingBlocks func() harness.TestNetworkDriver) {

			var mostRecentTxHash primitives.Sha256

			// generate some blocks with state
			contract := network.BenchmarkTokenContract()
			for i := 0; i < transfers; i++ {
				_, txHash := contract.Transfer(ctx, 0, transferAmount, 0, 1)
				mostRecentTxHash = txHash
			}

			network.WaitForTransactionInState(ctx, mostRecentTxHash)

			network = restartPreservingBlocks()
			contract = network.BenchmarkTokenContract()

			// wait for the most recent block height with transactions to reach state storage:
			network.WaitForTransactionInState(ctx, mostRecentTxHash)

			// verify state in both nodes
			balanceNode0 := contract.GetBalance(ctx, 0, 1)
			balanceNode1 := contract.GetBalance(ctx, 1, 1)

			require.EqualValues(t, totalAmount, balanceNode0, "expected transfers to reflect in leader state")
			require.EqualValues(t, totalAmount, balanceNode1, "expected transfers to reflect in non leader state")
		})

}
