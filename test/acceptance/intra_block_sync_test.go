package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestIntraBlockSyncTransactionPool(t *testing.T) {
	t.Skip()
	var aCommittedTxBuilder *builders.TransactionBuilder
	harness.Network(t).
		AllowingErrors(
			"leader failed to save block to storage",              // (block already in storage, skipping) TODO investigate and explain, or fix and remove expected error
			"internal-node sync to consensus algo failed",            //TODO investigate and explain, or fix and remove expected error
			"all consensus 0 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
			"all consensus 1 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
		).
		WithSetup(func(ctx context.Context, network harness.TestNetworkDriver) {
			for i := 1; i <= 10; i++ {
				aCommittedTxBuilder = builders.TransferTransaction().WithAmountAndTargetAddress(uint64(10*i), builders.AddressForEd25519SignerForTests(6))
				blockPair := builders.BenchmarkConsensusBlockPair().
					WithHeight(primitives.BlockHeight(i)).
					WithTransaction(aCommittedTxBuilder.Build()).
					Build()
				network.BlockPersistence(0).WriteNextBlock(blockPair)
			}
		}).Start(func(ctx context.Context, network harness.TestNetworkDriver) {

		// Wait for state storage to sync to block height 15
		if err := network.BlockPersistence(0).GetBlockTracker().WaitForBlock(ctx, 10); err != nil {
			t.Errorf("waiting for block on node 0 failed: %s", err)
		}

		ctx, _ = context.WithTimeout(ctx, 1 * time.Second)

		// Resend an already committed transaction to Leader
		leaderTxResponse := <- network.SendTransaction(ctx, aCommittedTxBuilder.Builder(), 0)
		require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, leaderTxResponse.TransactionStatus())

		// Resend an already committed transaction to Non-Leader
		nonLeaderTxResponse := <- network.SendTransaction(ctx, aCommittedTxBuilder.Builder(), 1)
		require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, nonLeaderTxResponse.TransactionStatus())
	})
}


func TestIntraBlockSyncState(t *testing.T) {
	t.Skip()
	harness.Network(t).
		AllowingErrors(
			"leader failed to save block to storage",              // (block already in storage, skipping) TODO investigate and explain, or fix and remove expected error
			"internal-node sync to consensus algo failed",         //TODO investigate and explain, or fix and remove expected error
			"all consensus 0 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
			"all consensus 1 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
		).
		WithSetup(func(ctx context.Context, network harness.TestNetworkDriver) {
			for i := 1; i <= 10; i++ {
				blockPair := builders.BenchmarkConsensusBlockPair().
					WithHeight(primitives.BlockHeight(i)).
					WithTransaction(builders.TransferTransaction().WithAmountAndTargetAddress(uint64(10), builders.AddressForEd25519SignerForTests(6)).Build()).
					Build()
				network.BlockPersistence(0).WriteNextBlock(blockPair)
			}
		}).Start(func(ctx context.Context, network harness.TestNetworkDriver) {

		// Wait for state storage to sync to block height 15
		if err := network.BlockPersistence(0).GetBlockTracker().WaitForBlock(ctx, 10); err != nil {
			t.Errorf("waiting for block on node 0 failed: %s", err)
		}

		targetAddress := builders.AddressForEd25519SignerForTests(6)
		getBalance := builders.GetBalanceTransaction().WithTargetAddress(targetAddress).Builder().Transaction

		require.EqualValues(t, 100, network.CallMethod(ctx, getBalance, 0), "expected transfers to reflect in leader state")
		require.EqualValues(t, 100, network.CallMethod(ctx, getBalance, 1), "expected transfers to reflect in non leader state")
	})
}