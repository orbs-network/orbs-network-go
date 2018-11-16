package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
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
		leaderTxResponse, ok := <-network.SendTransaction(ctx, txBuilders[0].Builder(), 0)
		require.True(t, ok)
		require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, leaderTxResponse.TransactionStatus())

		// Resend an already committed transaction to Non-Leader
		nonLeaderTxResponse, ok := <-network.SendTransaction(ctx, txBuilders[0].Builder(), 1)
		require.True(t, ok)
		require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, nonLeaderTxResponse.TransactionStatus())
	})
}

func TestInternalBlockSync_StateStorage(t *testing.T) {

	const transferAmount = 10
	const transfers = 10
	const totalAmount = transfers * transferAmount

	harness.Network(t).
		AllowingErrors(
			"leader failed to save block to storage",              // (block already in storage, skipping) TODO investigate and explain, or fix and remove expected error
			"internal-node sync to consensus algo failed",         //TODO Remove this once internal node sync is implemented
			"all consensus 0 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
			"all consensus 1 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
		).
		Start(func(ctx context.Context, builderNetwork harness.TestNetworkDriver) {

			contract := builderNetwork.GetBenchmarkTokenContract()
			var topBlock primitives.BlockHeight
			for i := 0; i < transfers; i++ {
				txRes := <- contract.SendTransfer(ctx,0, transferAmount, 0,1 )
				require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, txRes.TransactionStatus())
				topBlock = txRes.BlockHeight()
			}
			blockPairContainers, _,_,err := builderNetwork.BlockPersistence(0).GetBlocks(0, topBlock)
			require.NoError(t, err)

			harness.Network(t).
				AllowingErrors(
					"leader failed to save block to storage",              // (block already in storage, skipping) TODO investigate and explain, or fix and remove expected error
					"internal-node sync to consensus algo failed",         //TODO Remove this once internal node sync is implemented
					"all consensus 0 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
					"all consensus 1 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
				).
				WithSetup(func(ctx context.Context, network harness.TestNetworkDriver) {
					for _, bpc := range blockPairContainers {
						err := network.BlockPersistence(0).WriteNextBlock(bpc)
						require.NoError(t, err)
					}
				}).Start(func(ctx context.Context, network harness.TestNetworkDriver) {

				// Wait for state storage to sync both nodes to block height 10
				network.BlockPersistence(0).GetBlockTracker().WaitForBlock(ctx, topBlock)
				network.BlockPersistence(1).GetBlockTracker().WaitForBlock(ctx, topBlock)

				contract = network.GetBenchmarkTokenContract()

				// Read state entry from leader node
				leaderBalance := <- contract.CallGetBalance(ctx,0, 1)
				require.EqualValues(t, totalAmount, leaderBalance, "expected transfers to reflect in leader state")

				// Read state entry from non leader node
				nonLeaderBalance := <- contract.CallGetBalance(ctx,1, 1)
				require.EqualValues(t, totalAmount, nonLeaderBalance, "expected transfers to reflect in non leader state")
			})
		})
}
