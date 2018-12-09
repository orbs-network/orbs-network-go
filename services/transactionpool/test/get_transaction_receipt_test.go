package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGetTransactionReceiptFromPendingPoolAndCommittedPool(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(ctx)
		h.ignoringForwardMessages()

		tx1 := builders.Transaction().Build()
		tx2 := builders.Transaction().Build()
		h.addNewTransaction(ctx, tx1)
		h.addNewTransaction(ctx, tx2)

		h.assumeBlockStorageAtHeight(1)
		h.ignoringTransactionResults()
		h.reportTransactionsAsCommitted(ctx, tx2)

		out, err := h.txpool.GetCommittedTransactionReceipt(ctx, &services.GetCommittedTransactionReceiptInput{
			Txhash: digest.CalcTxHash(tx1.Transaction()),
		})

		require.NoError(t, err)
		require.Equal(t, protocol.TRANSACTION_STATUS_PENDING, out.TransactionStatus, "did not return expected status")
		require.Equal(t, h.lastBlockTimestamp, out.BlockTimestamp, "did not return expected timestamp")
		require.Equal(t, h.lastBlockHeight, out.BlockHeight, "did not return expected block height")

		tsOfCommittedTx := h.lastBlockTimestamp
		heightOfCommittedTx := h.lastBlockHeight
		h.goToBlock(ctx, 5, tsOfCommittedTx+100000)

		tx2hash := digest.CalcTxHash(tx2.Transaction())
		out, err = h.txpool.GetCommittedTransactionReceipt(ctx, &services.GetCommittedTransactionReceiptInput{
			Txhash: tx2hash,
		})

		require.NoError(t, err)
		require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, out.TransactionStatus, "did not return expected status")
		require.Equal(t, tx2hash, out.TransactionReceipt.Txhash(), "did not return expected receipt")
		require.Equal(t, tsOfCommittedTx, out.BlockTimestamp, "did not return expected timestamp")
		require.Equal(t, heightOfCommittedTx, out.BlockHeight, "did not return expected block height")

	})
}

func TestGetTransactionReceiptWhenTransactionNotFound(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(ctx)

		out, err := h.txpool.GetCommittedTransactionReceipt(ctx, &services.GetCommittedTransactionReceiptInput{
			Txhash: digest.CalcTxHash(builders.Transaction().Build().Transaction()),
		})

		require.NoError(t, err)
		require.Equal(t, protocol.TRANSACTION_STATUS_NO_RECORD_FOUND, out.TransactionStatus, "did not return expected status")
	})
}

func TestGetTransactionReceiptWhenTimestampAheadOfNodeTime(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(ctx)

		out, err := h.txpool.GetCommittedTransactionReceipt(ctx, &services.GetCommittedTransactionReceiptInput{
			TransactionTimestamp: primitives.TimestampNano(time.Now().Add(h.config.TransactionPoolFutureTimestampGraceTimeout() + 1*time.Minute).UnixNano()),
		})

		require.NoError(t, err)
		require.Equal(t, protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_AHEAD_OF_NODE_TIME, out.TransactionStatus, "did not return expected status")
	})
}
