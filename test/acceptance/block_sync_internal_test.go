package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInternalBlockSync_TransactionPool(t *testing.T) {

	blockCount := primitives.BlockHeight(10)
	txBuilders := make([]*builders.TransactionBuilder, blockCount)
	for i := 0; i < int(blockCount); i++ {
		txBuilders[i] = builders.TransferTransaction().WithAmountAndTargetAddress(uint64(i+1)*10, builders.AddressForEd25519SignerForTests(6))
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
					WithReceiptsForTransactions().
					WithHeight(i).
					Build()
				network.BlockPersistence(0).WriteNextBlock(blockPair)
			}
		}).Start(func(ctx context.Context, network harness.TestNetworkDriver) {

		lastTx := txBuilders[len(txBuilders)-1].Build().Transaction()
		waitForTransactionStatusCommitted(network, ctx, lastTx, 0)
		waitForTransactionStatusCommitted(network, ctx, lastTx, 1)

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

func waitForTransactionStatusCommitted(network harness.TestNetworkDriver, ctx context.Context, lastTx *protocol.Transaction, nodeIndex int) {
	var txStatusOut *services.GetTransactionStatusOutput
	for txStatusOut == nil || txStatusOut.ClientResponse.TransactionStatus() != protocol.TRANSACTION_STATUS_COMMITTED {
		txStatusOut, _ = network.PublicApi(nodeIndex).GetTransactionStatus(ctx, &services.GetTransactionStatusInput{
			ClientRequest: (&client.GetTransactionStatusRequestBuilder{
				TransactionTimestamp: 0,
				Txhash:               digest.CalcTxHash(lastTx),
			}).Build(),
		})
	}
}

func TestInternalBlockSync_StateStorage(t *testing.T) {

	const transferAmount = 10
	const transfers = 10
	const totalAmount = transfers * transferAmount

	var blockPairContainers []*protocol.BlockPairContainer
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
				txRes := <-contract.SendTransfer(ctx, 0, transferAmount, 0, 1)
				require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, txRes.TransactionStatus())
				topBlock = txRes.BlockHeight()
			}
			bpcs, _, _, err := builderNetwork.BlockPersistence(0).GetBlocks(1, topBlock+1)
			require.True(t, len(bpcs) >= transfers)
			require.NoError(t, err)
			blockPairContainers = bpcs
		})

		harness.Network(t).
			AllowingErrors(
				"leader failed to save block to storage",              // (block already in storage, skipping) TODO investigate and explain, or fix and remove expected error
				"internal-node sync to consensus algo failed",         //TODO Remove this once internal node sync is implemented
				"all consensus 0 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
				"all consensus 1 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
			).
			WithSetup(func(ctx context.Context, network harness.TestNetworkDriver) {
				// inject blocks from builder network
				for _, bpc := range blockPairContainers {
					err := network.BlockPersistence(0).WriteNextBlock(bpc)
					require.NoError(t, err)
				}
			}).Start(func(ctx context.Context, network harness.TestNetworkDriver) {

			// wait for top block to propagate to state in both nodes
			var topTxHash primitives.Sha256
			for _, bpc := range blockPairContainers {
				if len(bpc.ResultsBlock.TransactionReceipts) > 0 {
					topTxHash = bpc.ResultsBlock.TransactionReceipts[0].Txhash()
				}
			}
			network.WaitForTransactionInNodeState(ctx, topTxHash, 0)
			network.WaitForTransactionInNodeState(ctx, topTxHash, 1)

			contract := network.GetBenchmarkTokenContract()

			// verify state in both nodes
			balanceNode0 := <- contract.CallGetBalance(ctx,0, 1)
			balanceNode1 := <- contract.CallGetBalance(ctx,1, 1)

			require.EqualValues(t, totalAmount, balanceNode0, "expected transfers to reflect in leader state")
			require.EqualValues(t, totalAmount, balanceNode1, "expected transfers to reflect in non leader state")
		})

}
