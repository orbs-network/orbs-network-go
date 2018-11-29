package acceptance

import (
	"bytes"
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	. "github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

var STATUS_COMMITTED_OR_DUPLICATE = []TransactionStatus{TRANSACTION_STATUS_COMMITTED, TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING}
var STATUS_DUPLICATE = []TransactionStatus{TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING}

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
		response0 := network.SendTransaction(ctx, builders.TransferTransaction().WithTimestamp(ts).Builder(), 1)
		response1 := network.SendTransaction(ctx, builders.TransferTransaction().WithTimestamp(ts).Builder(), 1)

		require.Contains(t, STATUS_COMMITTED_OR_DUPLICATE, response0.TransactionStatus(), "second transaction should be accepted into the pool and committed or rejected as duplidate")
		require.Contains(t, STATUS_DUPLICATE, response1.TransactionStatus(), "third transaction should be rejected as a duplicate")

		require.True(t, response0.BlockHeight() <= response1.BlockHeight(), "second response must reference a later block height than first")

		requireTxCommittedOnce(ctx, t, response1.BlockHeight()+5, network, response0.TransactionReceipt().Txhash())

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

func TestSendSameTransactionFastTwiceToLeader(t *testing.T) {
	harness.Network(t).AllowingErrors(
		"error adding transaction to pending pool",
		"error adding forwarded transaction to pending pool",
		"error sending transaction",
		"transaction rejected: TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING",
	).Start(func(ctx context.Context, network harness.TestNetworkDriver) {

		ts := time.Now()
		contract := network.GetBenchmarkTokenContract()
		contract.DeployBenchmarkToken(ctx, 1)

		//TODO this should be the same builder, but membuffers has a stability bug preventing re-usage of builders
		tx1 := builders.TransferTransaction().WithTimestamp(ts).Builder()
		tx2 := builders.TransferTransaction().WithTimestamp(ts).Builder()

		network.SendTransactionInBackground(ctx, tx1, 0)
		secondAttemptResponse := network.SendTransaction(ctx, tx2, 0)

		// A race condition here makes three possible outcomes:
		// - secondAttemptResponse is nil, which means an error was returned // TODO understand under what circumstances an error here is ok
		// - a response with TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING status was received if tx is not yet committed
		// - a response with TRANSACTION_STATUS_COMMITTED status was received if tx is already committed
		if secondAttemptResponse != nil {
			t.Logf("received status %s in second SendTransaction", secondAttemptResponse.TransactionStatus().String())
			require.Contains(t, STATUS_COMMITTED_OR_DUPLICATE, secondAttemptResponse.TransactionStatus(), "second attempt must return ALREADY_PENDING or COMMITTED status")

			requireTxCommittedOnce(ctx, t, secondAttemptResponse.BlockHeight()+5, network, digest.CalcTxHash(tx1.Build().Transaction()))
		}
	})
}
