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

func TestInternalBlockSync_TransactionPool(t *testing.T) {

	blockCount := primitives.BlockHeight(10)
	txBuilders := make([]*builders.TransactionBuilder, blockCount)
	for i := 0; i < int(blockCount); i++ {
		txBuilders[i] = builders.TransferTransaction().WithAmountAndTargetAddress(uint64(i)*10, builders.AddressForEd25519SignerForTests(6))
	}

	harness.Network(t).
		AllowingErrors(
			"leader failed to save block to storage",              // (block already in storage, skipping) TODO investigate and explain, or fix and remove expected error
			"internal-node sync to consensus algo failed",         //TODO Remove this once internal node sync is implemented
			"all consensus 0 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
			"all consensus 1 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
		).
		WithSetup(func(ctx context.Context, network harness.TestNetworkDriver) {
			for i := primitives.BlockHeight(1); i <= blockCount; i++ {
				blockPair := builders.BenchmarkConsensusBlockPair().
					WithTransaction(txBuilders[i-1].Build()).
					WithHeight(i).
					Build()
				network.BlockPersistence(0).WriteNextBlock(blockPair)
			}
		}).Start(func(ctx context.Context, network harness.TestNetworkDriver) {

		// Wait for state storage to sync both nodes to block height 10
		network.BlockPersistence(0).GetBlockTracker().WaitForBlock(ctx, blockCount)
		network.BlockPersistence(1).GetBlockTracker().WaitForBlock(ctx, blockCount)

		// Resend an already committed transaction to Leader
		ctx, _ = context.WithTimeout(ctx, 1*time.Second)
		leaderTxResponse := <-network.SendTransaction(ctx, txBuilders[0].Builder(), 0)
		require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, leaderTxResponse.TransactionStatus())

		// Resend an already committed transaction to Non-Leader
		nonLeaderTxResponse := <-network.SendTransaction(ctx, txBuilders[0].Builder(), 1)
		require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, nonLeaderTxResponse.TransactionStatus())
	})
}

func TestInternalBlockSync_StateStorage(t *testing.T) {


	harness.Network(t).
		AllowingErrors(
			"leader failed to save block to storage",              // (block already in storage, skipping) TODO investigate and explain, or fix and remove expected error
			"internal-node sync to consensus algo failed",         //TODO Remove this once internal node sync is implemented
			"all consensus 0 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
			"all consensus 1 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
		).
		Start(func(ctx context.Context, builderNetwork harness.TestNetworkDriver) {

			ctx, _ = context.WithTimeout(ctx, 1*time.Second)

			contract := builderNetwork.GetBenchmarkTokenContract()
			txRes1 := <- contract.SendTransfer(ctx,0, 10, 0,1 )
			txRes2 := <- contract.SendTransfer(ctx,0, 10, 0,1 )

			require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, txRes1.TransactionStatus())
			require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, txRes2.TransactionStatus())

			targetBlockHeight := txRes2.BlockHeight()

			harness.Network(t).
				AllowingErrors(
					"leader failed to save block to storage",              // (block already in storage, skipping) TODO investigate and explain, or fix and remove expected error
					"internal-node sync to consensus algo failed",         //TODO Remove this once internal node sync is implemented
					"all consensus 0 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
					"all consensus 1 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
				).
				WithSetup(func(ctx context.Context, network harness.TestNetworkDriver) {
					containers, _,_,err := builderNetwork.BlockPersistence(0).GetBlocks(0, targetBlockHeight)
					require.NoError(t, err)
					for _, bpc := range containers {
						err := network.BlockPersistence(0).WriteNextBlock(bpc)
						require.NoError(t, err)
					}
				}).Start(func(ctx context.Context, network harness.TestNetworkDriver) {

				// Wait for state storage to sync both nodes to block height 10
				network.BlockPersistence(0).GetBlockTracker().WaitForBlock(ctx, targetBlockHeight)
				network.BlockPersistence(1).GetBlockTracker().WaitForBlock(ctx, targetBlockHeight)

				contract = network.GetBenchmarkTokenContract()
				leaderBalance := <- contract.CallGetBalance(ctx,0, 1)
				nonLeaderBalance := <- contract.CallGetBalance(ctx,1, 1)

				require.EqualValues(t, 20, nonLeaderBalance, "expected transfers to reflect in non leader state")
				require.EqualValues(t, 20, leaderBalance, "expected transfers to reflect in leader state")
			})
		})
}
