package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
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
		require.True(t, waitForTransactionStatusCommitted(ctx, network, digest.CalcTxHash(lastTx), 0),
			"expected tx to be committed to leader tx pool")
		require.True(t, waitForTransactionStatusCommitted(ctx, network, digest.CalcTxHash(lastTx), 1),
			"expected tx to be committed to non leader tx pool")

		// Resend an already committed transaction to Leader
		leaderTxResponse := network.SendTransaction(ctx, txBuilders[0].Builder(), 0)
		require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, leaderTxResponse.TransactionStatus())

		// Resend an already committed transaction to Non-Leader
		nonLeaderTxResponse := network.SendTransaction(ctx, txBuilders[0].Builder(), 1)
		require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, nonLeaderTxResponse.TransactionStatus())
	})
}

func waitForTransactionStatusCommitted(ctx context.Context, network harness.TestNetworkDriver, txHash primitives.Sha256, nodeIndex int) bool {
	return test.Eventually(5*time.Second, func() bool {
		txStatusOut, err := network.PublicApi(nodeIndex).GetTransactionStatus(ctx, &services.GetTransactionStatusInput{
			ClientRequest: (&client.GetTransactionStatusRequestBuilder{
				TransactionTimestamp: 0,
				Txhash:               txHash,
			}).Build(),
		})
		if err != nil {
			return false
		}
		return txStatusOut.ClientResponse.TransactionStatus() == protocol.TRANSACTION_STATUS_COMMITTED
	})
}

func TestInternalBlockSync_StateStorage(t *testing.T) {
	const transferAmount = 10
	const transfers = 10
	const totalAmount = transfers * transferAmount

	containersChan := make(chan []*protocol.BlockPairContainer, 1)
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
				txRes := contract.SendTransfer(ctx, 0, transferAmount, 0, 1)
				require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, txRes.TransactionStatus())
				topBlock = txRes.BlockHeight()
			}
			bpcs, _, _, err := builderNetwork.BlockPersistence(0).GetBlocks(1, topBlock+1)
			require.True(t, len(bpcs) >= transfers)
			require.NoError(t, err)
			containersChan <- bpcs
		})

	blockPairContainers := <-containersChan
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
				_, err := network.BlockPersistence(0).WriteNextBlock(bpc)
				require.NoError(t, err)
			}
		}).Start(func(ctx context.Context, network harness.TestNetworkDriver) {

		// wait for the most recent block height with transactions to reach state storage:
		// TODO if we can wait for state storage to reach a block height we don't need this ugly loop
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
		balanceNode0 := contract.CallGetBalance(ctx, 0, 1)
		balanceNode1 := contract.CallGetBalance(ctx, 1, 1)

		require.EqualValues(t, totalAmount, balanceNode0, "expected transfers to reflect in leader state")
		require.EqualValues(t, totalAmount, balanceNode1, "expected transfers to reflect in non leader state")
	})

}
