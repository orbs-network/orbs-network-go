package acceptance

import (
	"bytes"
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	. "github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

var STATUS_COMMITTED_OR_PENDING_OR_DUPLICATE = []TransactionStatus{TRANSACTION_STATUS_COMMITTED, TRANSACTION_STATUS_PENDING, TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING}
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
		response0, txHash := network.SendTransaction(ctx, builders.TransferTransaction().WithTimestamp(ts).Builder(), 1)
		response1, _ := network.SendTransaction(ctx, builders.TransferTransaction().WithTimestamp(ts).Builder(), 1)

		require.Contains(t, STATUS_COMMITTED_OR_PENDING_OR_DUPLICATE, response0.TransactionStatus(), "second transaction should be accepted into the pool or rejected as duplidate")
		require.Contains(t, STATUS_DUPLICATE, response1.TransactionStatus(), "third transaction should be rejected as a duplicate")

		requireTxCommittedOnce(ctx, t, network, txHash)
	})
}

func requireTxCommittedOnce(ctx context.Context, t *testing.T, network harness.TestNetworkDriver, txHash primitives.Sha256) {
	// wait for the tx to be seen as committed in state
	network.WaitForTransactionInState(ctx, txHash)
	txHeight, err := network.BlockPersistence(0).GetNumBlocks()
	require.NoError(t, err)

	// wait for 5 more blocks to be committed
	height := txHeight + 5
	err = network.BlockPersistence(0).GetBlockTracker().WaitForBlock(ctx, height)
	require.NoError(t, err, "expected to reach target block height before proceeding with test")

	// count receipts for txHash in leader block storage
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

		// this should be the same builder, but membuffers is not thread-safe for concurrent builds on same builder
		tx1 := builders.TransferTransaction().WithTimestamp(ts).Builder()
		tx2 := builders.TransferTransaction().WithTimestamp(ts).Builder()

		network.SendTransactionInBackground(ctx, tx1, 0)
		secondAttemptResponse, txHash := network.SendTransaction(ctx, tx2, 0)

		t.Logf("received status %s in second SendTransaction", secondAttemptResponse.TransactionStatus().String())
		require.Contains(t, STATUS_COMMITTED_OR_DUPLICATE, secondAttemptResponse.TransactionStatus(), "second attempt must return COMMITTED or DUPLICATE status")

		requireTxCommittedOnce(ctx, t, network, txHash)
	})
}
