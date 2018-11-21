package acceptance

import (
	"bytes"
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSendSameTransactionMoreThanOnce1(t *testing.T) {
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

		// second transaction will be accepted into the pool and committed by the internode sync
		require.EqualValues(t, protocol.TRANSACTION_STATUS_COMMITTED, response0.TransactionStatus())
		// third transaction should be rejected as a duplicate
		require.EqualValues(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, response1.TransactionStatus())

		// both responses must indicate the same block height
		require.EqualValues(t, response0.BlockHeight(), response1.BlockHeight())

		// wait for 5 more blocks to be closed, and check the blockchain for duplicate tx entries
		farBlockHeight := response1.BlockHeight() + 5
		err := network.BlockPersistence(0).GetBlockTracker().WaitForBlock(ctx, farBlockHeight)
		require.NoError(t, err)

		receiptCount := 0
		blocks, _, _, err := network.BlockPersistence(0).GetBlocks(1, farBlockHeight)
		require.NoError(t, err)
		require.Len(t, blocks, int(farBlockHeight))
		for _, block := range blocks {
			for _, r := range block.ResultsBlock.TransactionReceipts {
				if bytes.Equal(r.Txhash(), response0.TransactionReceipt().Txhash()) {
					receiptCount++
				}
			}
		}
		require.Equal(t, 1, receiptCount)

	})
}

// TODO enable this test once we find a way to allow the error that is thrown when TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING happens
func TestSendSameTransactionMoreThanOnce2(t *testing.T) {
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
		// - secondAttemptResponse is nil, which means an error was returned
		// - a response with TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING status was received if tx is not yet committed
		// - a response with TRANSACTION_STATUS_COMMITTED status was received if tx is already committed
		if secondAttemptResponse != nil {
			require.True(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING == secondAttemptResponse.TransactionStatus() ||
				protocol.TRANSACTION_STATUS_COMMITTED == secondAttemptResponse.TransactionStatus())
		}
	})
}
