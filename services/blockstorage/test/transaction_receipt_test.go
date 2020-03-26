// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/crypto-lib-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestReturnTransactionReceiptIfTransactionNotFound(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		harness := newBlockStorageHarness(parent).
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

func TestReturnTransactionReceipt(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		harness := newBlockStorageHarness(parent).
			withSyncBroadcast(1).
			withCommitStateDiff(1).
			withValidateConsensusAlgos(1).
			start(ctx)

		txQueryGrace := harness.config.BlockStorageTransactionReceiptQueryTimestampGrace()
		txExpirationWnd := harness.config.TransactionExpirationWindow()

		// block1: txs with current time but block timestamps in the past before grace
		block1 := builders.BlockPair().WithHeight(1).WithTransactions(10).WithReceiptsForTransactions().WithTimestampAheadBy(-1 * (txQueryGrace + time.Second)).Build()
		harness.commitBlock(ctx, block1)

		require.True(t, block1.TransactionsBlock.SignedTransactions[3].Transaction().Timestamp() > block1.TransactionsBlock.Header.Timestamp()+primitives.TimestampNano(txQueryGrace.Nanoseconds()), "expected block to be in the past")

		// block closed with ts in the past within grace
		block2 := builders.BlockPair().WithHeight(2).WithTransactions(10).WithReceiptsForTransactions().WithTimestampAheadBy(txQueryGrace / -2).Build()
		harness.commitBlock(ctx, block2)

		// block closed with ts in the expiration window
		block3 := builders.BlockPair().WithHeight(3).WithTransactions(10).WithReceiptsForTransactions().WithTimestampAheadBy(txExpirationWnd / 2).Build()
		harness.commitBlock(ctx, block3)

		// block closed with ts past the expiration window but within grace
		block4 := builders.BlockPair().WithHeight(4).WithTransactions(10).WithReceiptsForTransactions().WithTimestampAheadBy(txExpirationWnd + txQueryGrace/2).Build()
		harness.commitBlock(ctx, block4)

		// block closed with ts past both the expiration window and grace period
		block5 := builders.BlockPair().WithHeight(5).WithTransactions(10).WithReceiptsForTransactions().WithTimestampAheadBy(txExpirationWnd + txQueryGrace + 1).Build()
		harness.commitBlock(ctx, block5)

		requireTransactionFoundInBlock(ctx, t, harness, block2)
		requireTransactionFoundInBlock(ctx, t, harness, block3)
		requireTransactionFoundInBlock(ctx, t, harness, block4)

		requireTransactionNotFoundInBlock(ctx, t, harness, block1, block5)
		requireTransactionNotFoundInBlock(ctx, t, harness, block5, block5)
	})
}

func requireTransactionFoundInBlock(ctx context.Context, t *testing.T, harness *harness, block *protocol.BlockPairContainer) {
	txHash, out, err := searchForTx(block.TransactionsBlock.SignedTransactions[3].Transaction(), harness, ctx)

	require.NoError(t, err, "receipt should be found in this flow")
	require.NotNil(t, out.TransactionReceipt, "receipt should be found in this flow")
	require.EqualValues(t, txHash, out.TransactionReceipt.Txhash(), "receipt should have the tx hash we looked for")
	require.EqualValues(t, block.ResultsBlock.Header.BlockHeight(), out.BlockHeight, "receipt should have the block height of the block containing the transaction")
	require.EqualValues(t, block.ResultsBlock.Header.Timestamp(), out.BlockTimestamp, "receipt should have the timestamp of the block containing the transaction")
}

func requireTransactionNotFoundInBlock(ctx context.Context, t *testing.T, harness *harness, block *protocol.BlockPairContainer, lastCommittedBlock *protocol.BlockPairContainer) {
	_, out, err := searchForTx(block.TransactionsBlock.SignedTransactions[3].Transaction(), harness, ctx)

	require.NoError(t, err, "receipt should be found in this flow")
	require.Nil(t, out.TransactionReceipt, "receipt should not be found in this flow")
	require.EqualValues(t, lastCommittedBlock.ResultsBlock.Header.BlockHeight(), out.BlockHeight, "result should have the currently top committed block height")
	require.EqualValues(t, lastCommittedBlock.ResultsBlock.Header.Timestamp(), out.BlockTimestamp, "result should have the timestamp of the currently top committed block")
}

func searchForTx(tx *protocol.Transaction, harness *harness, ctx context.Context) (primitives.Sha256, *services.GetTransactionReceiptOutput, error) {
	// taking a transaction at 'random' (they were created at random)
	txHash := digest.CalcTxHash(tx)
	// the block timestamp is just a couple of nanos ahead of the transactions, which is inside the grace
	out, err := harness.blockStorage.GetTransactionReceipt(ctx, &services.GetTransactionReceiptInput{
		Txhash:               txHash,
		TransactionTimestamp: tx.Timestamp(),
	})
	return txHash, out, err
}
