// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestReturnTransactionReceiptIfTransactionNotFound(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness(t).
			withSyncBroadcast(1).
			withCommitStateDiff(1).
			withValidateConsensusAlgos(1).
			start(ctx)

		block := builders.BlockPair().Build()
		harness.commitBlock(ctx, block)

		out, err := harness.blockStorage.GetTransactionReceipt(ctx, &services.GetTransactionReceiptInput{
			Txhash:               []byte("will-not-be-found"),
			TransactionTimestamp: primitives.TimestampNano(time.Now().UnixNano()),
		})

		require.NoError(t, err, "transaction not found happy flow")
		require.Nil(t, out.TransactionReceipt, "represents an empty receipt")
		require.EqualValues(t, 1, out.BlockHeight, "last block height")
		require.EqualValues(t, block.ResultsBlock.Header.Timestamp(), out.BlockTimestamp, "last block timestamp")
	})
}

// TODO(v1) return transaction receipt while the transaction timestamp is in the future (and too far ahead to be in the grace

func TestReturnTransactionReceipt(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness(t).
			withSyncBroadcast(1).
			withCommitStateDiff(1).
			withValidateConsensusAlgos(1).
			start(ctx)

		block := builders.BlockPair().WithHeight(1).WithTransactions(10).WithReceiptsForTransactions().WithTimestampNow().Build()
		harness.commitBlock(ctx, block)

		// it will be similar data transactions, but with different time stamps (and hashes..)
		block2 := builders.BlockPair().WithHeight(2).WithTransactions(10).WithReceiptsForTransactions().WithTimestampNow().Build()
		harness.commitBlock(ctx, block2)

		// taking a transaction at 'random' (they were created at random)
		tx := block.TransactionsBlock.SignedTransactions[3].Transaction()
		txHash := digest.CalcTxHash(tx)

		// the block timestamp is just a couple of nanos ahead of the transactions, which is inside the grace
		out, err := harness.blockStorage.GetTransactionReceipt(ctx, &services.GetTransactionReceiptInput{
			Txhash:               txHash,
			TransactionTimestamp: tx.Timestamp(),
		})

		require.NoError(t, err, "receipt should be found in this flow")
		require.NotNil(t, out.TransactionReceipt, "receipt should be found in this flow")
		require.EqualValues(t, txHash, out.TransactionReceipt.Txhash(), "receipt should have the tx hash we looked for")
		require.EqualValues(t, 1, out.BlockHeight, "receipt should have the block height of the block containing the transaction")
		require.EqualValues(t, block.ResultsBlock.Header.Timestamp(), out.BlockTimestamp, "receipt should have the timestamp of the block containing the transaction")
	})
}

// TODO(v1) return transaction receipt while the transaction timestamp is outside the grace (regular)
// TODO(v1) return transaction receipt while the transaction timestamp is at the expire window
// TODO(v1) return transaction receipt while the transaction timestamp is at the expire window and within the grace
