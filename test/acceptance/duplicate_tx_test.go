package acceptance

import (
	"bytes"
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSendSameTransactionFastToTwoNodes(t *testing.T) {
	harness.Network(t).AllowingErrors(
		"error adding transaction to pending pool",
		"error adding forwarded transaction to pending pool",
		"error sending transaction",
	).Start(func(ctx context.Context, network harness.TestNetworkDriver) {
		ts := time.Now()

		contract := network.GetBenchmarkTokenContract()
		contract.DeployBenchmarkToken(ctx, 1)

		// send three identical transactions to two nodes
		network.SendTransactionInBackground(ctx, builders.TransferTransaction().WithTimestamp(ts).Builder(), 0)
		response0 := <- network.SendTransaction(ctx, builders.TransferTransaction().WithTimestamp(ts).Builder(), 1)
		response1 := <- network.SendTransaction(ctx, builders.TransferTransaction().WithTimestamp(ts).Builder(), 1)

		require.EqualValues(t, protocol.TRANSACTION_STATUS_COMMITTED, response0.TransactionStatus(), "second transaction should be accepted into the pool and committed by the internode sync")
		require.EqualValues(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, response1.TransactionStatus(), "third transaction should be rejected as a duplicate")

		require.True(t, response0.BlockHeight() <= response1.BlockHeight(), "second response must reference a later block height than first")

		requireTxCommittedOnce( ctx, t, response1.BlockHeight() + 5, network, response0.TransactionReceipt().Txhash())

	})
}

func requireTxCommittedOnce(ctx context.Context, t *testing.T, height primitives.BlockHeight, network harness.TestNetworkDriver, txHash primitives.Sha256) {
	err := network.BlockPersistence(0).GetBlockTracker().WaitForBlock(ctx, height)
	require.NoError(t, err, "expected to reach target block height before proceeding with test")
	receiptCount := 0
	blocks, _, _, err := network.BlockPersistence(0).GetBlocks(1, height)
	require.NoError(t, err, "GetBlocks should return blocks")
	require.Len(t, blocks, int(height), "GetBlocks should return %d blocks", height)
	for _, block := range blocks {
		for _, r := range block.ResultsBlock.TransactionReceipts {
			if bytes.Equal(r.Txhash(), txHash) {
				receiptCount++
			}
		}
	}
	require.Equal(t, 1, receiptCount, "blocks should include tx exactly once")
}

// TODO enable this test once we find a way to allow the error that is thrown when TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING happens
func TestSendSameTransactionFastTwiceToLeader(t *testing.T) {
	t.Skip("disabled due to harness issue")

	harness.Network(t).AllowingErrors(
		"error adding transaction to pending pool",
		"error adding forwarded transaction to pending pool",
		"error sending transaction",
	).Start(func(ctx context.Context, network harness.TestNetworkDriver) {

		ts := time.Now()
		contract := network.GetBenchmarkTokenContract()
		contract.DeployBenchmarkToken(ctx, 1)

		network.SendTransactionInBackground(ctx, builders.TransferTransaction().WithTimestamp(ts).Builder(), 0)
		secondAttemptResponse := <- network.SendTransaction(ctx, builders.TransferTransaction().WithTimestamp(ts).Builder(), 0)

		// A race condition here makes three possible outcomes:
		// - secondAttemptResponse is nil, which means an error was returned // TODO understand under what circumstances an error here is ok
		// - a response with TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING status was received if tx is not yet committed
		// - a response with TRANSACTION_STATUS_COMMITTED status was received if tx is already committed
		if secondAttemptResponse != nil {
			require.True(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING == secondAttemptResponse.TransactionStatus() ||
				protocol.TRANSACTION_STATUS_COMMITTED == secondAttemptResponse.TransactionStatus(), "second attempt must ")
		}

		requireTxCommittedOnce( ctx, t, secondAttemptResponse.BlockHeight() + 5, network, secondAttemptResponse.TransactionReceipt().Txhash())

	})
}
