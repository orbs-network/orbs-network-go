// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCommitTransactionReceiptsRequestsNextBlockOnMismatch(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t).start(ctx)

		h.assumeBlockStorageAtHeight(0) // so that we report transactions for block 1
		out, err := h.reportTransactionsAsCommitted(ctx)
		require.NoError(t, err, "CommitTransactionReceipts returned an error when expecting next desired block height")
		require.EqualValues(t, 2, out.NextDesiredBlockHeight, "expected next desired block height to be 2")

		h.assumeBlockStorageAtHeight(3) // so that we report transactions for block 4
		out, err = h.reportTransactionsAsCommitted(ctx)
		require.NoError(t, err, "CommitTransactionReceipts returned an error when expecting next desired block height")
		require.EqualValues(t, 2, out.NextDesiredBlockHeight, "expected next desired block height to be 2")

		h.ignoringTransactionResults()

		require.NoError(t, h.verifyMocks())
	})
}

func TestCommitTransactionReceiptForTxThatWasNeverInPendingPool_ShouldCommitItAnyway(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t).start(ctx)
		tx := builders.TransferTransaction().Build()

		h.reportTransactionsAsCommitted(ctx, tx)

		output, err := h.getTxReceipt(ctx, tx)
		require.NoError(t, err, "could not get output for tx committed without adding it to pending pool")
		require.NotNil(t, output)
		require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED.String(), output.TransactionStatus.String(), "transaction was not committed")
		require.NotNil(t, output.TransactionReceipt, "transaction was not committed")

		require.NoError(t, h.verifyMocks(), "Mocks were not executed as planned")
	})
}

func TestCommitTransactionReceiptsIgnoresExpiredBlocks(t *testing.T) {
	t.Skipf("TODO(v1): ignore blocks with an expired timestamp")
}
