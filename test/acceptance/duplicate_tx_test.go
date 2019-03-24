// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"bytes"
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	. "github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

var STATUS_COMMITTED_OR_PENDING_OR_DUPLICATE = []TransactionStatus{TRANSACTION_STATUS_COMMITTED, TRANSACTION_STATUS_PENDING, TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING}
var STATUS_COMMITTED_OR_DUPLICATE = []TransactionStatus{TRANSACTION_STATUS_COMMITTED, TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING}
var STATUS_DUPLICATE = []TransactionStatus{TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING}

// LH: Use ControlledRandom (ctrlrnd.go) (in acceptance harness) to generate the initial RandomSeed and put it in LeanHelix's config
func TestSendSameTransactionFastToTwoNodes(t *testing.T) {
	newHarness().AllowingErrors(
		"error adding transaction to pending pool",
		"error adding forwarded transaction to pending pool",
		"error sending transaction",
	).Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
		ts := time.Now()

		network.DeployBenchmarkTokenContract(ctx, 1)

		// send three identical transactions to two nodes
		tx := builders.TransferTransaction().WithTimestamp(ts).Builder()
		identicalTx := builders.TransferTransaction().WithTimestamp(ts).Builder()

		require.EqualValues(t, tx.Build().Raw(), identicalTx.Build().Raw())

		go func(ctx context.Context) {
			response, txHash := network.SendTransaction(ctx, tx, 0)
			t.Log("node #0 send #1", response.TransactionStatus().String(), txHash.String())
		}(ctx)

		response0, txHash0 := network.SendTransaction(ctx, identicalTx, 1)
		response1, txHash1 := network.SendTransaction(ctx, identicalTx, 1)

		t.Log("node #1 send #1", response0.TransactionStatus().String(), txHash0)
		t.Log("node #1 send #2", response1.TransactionStatus().String(), txHash1)

		require.Equal(t, txHash0, txHash1, "expect same transactions to produce same txHash")

		require.Contains(t, STATUS_COMMITTED_OR_PENDING_OR_DUPLICATE, response0.TransactionStatus(), "second transaction should be accepted into the pool or rejected as duplidate")
		require.Contains(t, STATUS_DUPLICATE, response1.TransactionStatus(), "third transaction should be rejected as a duplicate")

		requireTxCommittedOnce(ctx, t, network, 0, txHash0)
		requireTxCommittedOnce(ctx, t, network, 1, txHash0)
	})
}

// LH: Use ControlledRandom (ctrlrnd.go) (in acceptance harness) to generate the initial RandomSeed and put it in LeanHelix's config
func TestSendSameTransactionFastTwiceToSameNode(t *testing.T) {
	newHarness().AllowingErrors(
		"error adding transaction to pending pool",
		"error adding forwarded transaction to pending pool",
		"error sending transaction",
		"transaction rejected: TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING",
	).Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {

		ts := time.Now()
		network.DeployBenchmarkTokenContract(ctx, 1)

		// send three identical transactions to two nodes
		tx := builders.TransferTransaction().WithTimestamp(ts).Builder()
		identicalTx := builders.TransferTransaction().WithTimestamp(ts).Builder()

		go func(ctx context.Context) {
			response, txHash := network.SendTransaction(ctx, tx, 0)
			t.Log("node #0 send #1", response.TransactionStatus().String(), txHash.String())
		}(ctx)

		secondAttemptResponse, txHash := network.SendTransaction(ctx, identicalTx, 0)
		t.Log("node #0 send #2", secondAttemptResponse.TransactionStatus().String(), txHash.String())

		require.Contains(t, STATUS_COMMITTED_OR_DUPLICATE, secondAttemptResponse.TransactionStatus(), "second attempt must return COMMITTED or DUPLICATE status")

		requireTxCommittedOnce(ctx, t, network, 0, txHash)
		requireTxCommittedOnce(ctx, t, network, 1, txHash)
	})
}

func requireTxCommittedOnce(ctx context.Context, t testing.TB, network *NetworkHarness, nodeIndex int, txHash primitives.Sha256) {
	// wait for the tx to be seen as committed in state
	network.WaitForTransactionInState(ctx, txHash)
	persistence := network.BlockPersistence(nodeIndex)

	txHeight, err := persistence.GetLastBlockHeight()
	require.NoError(t, err)

	// wait for 5 more blocks to be committed
	height := txHeight + 5
	err = persistence.GetBlockTracker().WaitForBlock(ctx, height)
	require.NoError(t, err, "expected to reach target block height before proceeding with test")

	// count receipts for txHash in block storage of node 0
	receiptCount := 0
	var blocks []*BlockPairContainer
	err = network.BlockPersistence(0).ScanBlocks(1, uint8(height), func(first primitives.BlockHeight, page []*BlockPairContainer) bool {
		blocks = page
		return false
	})
	require.NoError(t, err, "ScanBlocks should return blocks, instead got error %v", err)

	// TODO (v1) https://github.com/orbs-network/orbs-network-go/issues/837 do we want to keep this require? it may hide a bug in sync between ScanBlocks and WaitForBlock
	//require.Len(t, blocks, int(height), "ScanBlocks should return %d blocks, instead got %d", height, len(blocks))
	for _, block := range blocks {
		for _, r := range block.ResultsBlock.TransactionReceipts {
			if bytes.Equal(r.Txhash(), txHash) {
				receiptCount++
			}
		}
	}
	require.Equal(t, 1, receiptCount, "blocks should include tx exactly once")
}

func TestBlockTrackerAndScanBlocksStayInSync(t *testing.T) {
	newHarness().AllowingErrors(
		"error adding transaction to pending pool",
		"error adding forwarded transaction to pending pool",
		"error sending transaction",
		"transaction rejected: TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING",
	).Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {

		persistence := network.BlockPersistence(0)
		targetBlockHeight := 2
		err1 := persistence.GetBlockTracker().WaitForBlock(ctx, primitives.BlockHeight(targetBlockHeight))

		err2 := network.BlockPersistence(0).ScanBlocks(1, uint8(targetBlockHeight), func(first primitives.BlockHeight, page []*BlockPairContainer) bool {
			require.Len(t, page, targetBlockHeight)
			return false
		})
		require.NoError(t, err1)
		require.NoError(t, err2)
	})
}
