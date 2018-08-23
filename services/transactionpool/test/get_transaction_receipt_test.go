package test

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetTransactionReceiptFromPendingPoolAndCommittedPool(t *testing.T) {
	t.Parallel()
	h := newHarness()
	h.ignoringForwardMessages()

	tx1 := builders.Transaction().Build()
	tx2 := builders.Transaction().Build()
	h.addNewTransaction(tx1)
	h.addNewTransaction(tx2)

	h.assumeBlockStorageAtHeight(1)
	h.ignoringTransactionResults()
	h.reportTransactionsAsCommitted(tx2)

	out, err := h.txpool.GetCommittedTransactionReceipt(&services.GetCommittedTransactionReceiptInput{
		Txhash: digest.CalcTxHash(tx1.Transaction()),
	})

	require.NoError(t, err)
	require.Equal(t, protocol.TRANSACTION_STATUS_PENDING, out.TransactionStatus, "did not return expected status")
	require.Equal(t, h.lastBlockTimestamp, out.BlockTimestamp, "did not return expected timestamp")
	require.Equal(t, h.lastBlockHeight, out.BlockHeight, "did not return expected block height")

	tx2hash := digest.CalcTxHash(tx2.Transaction())
	out, err = h.txpool.GetCommittedTransactionReceipt(&services.GetCommittedTransactionReceiptInput{
		Txhash: tx2hash,
	})

	require.NoError(t, err)
	require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, out.TransactionStatus, "did not return expected status")
	require.Equal(t, tx2hash, out.TransactionReceipt.Txhash(), "did not return expected receipt")
}

func TestGetTransactionReceiptWhenTransactionNotFound(t *testing.T) {
	t.Parallel()
	h := newHarness()

	out, err := h.txpool.GetCommittedTransactionReceipt(&services.GetCommittedTransactionReceiptInput{
		Txhash: digest.CalcTxHash(builders.Transaction().Build().Transaction()),
	})

	require.NoError(t, err)
	require.Equal(t, protocol.TRANSACTION_STATUS_NO_RECORD_FOUND, out.TransactionStatus, "did not return expected status")

}
