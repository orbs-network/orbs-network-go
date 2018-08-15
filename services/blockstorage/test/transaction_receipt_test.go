package test

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestReturnTransactionReceiptIfTransactionNotFound(t *testing.T) {
	driver := NewDriver(t)
	driver.expectCommitStateDiff()

	block := builders.BlockPair().WithTimestampBloomFilter().Build()
	driver.commitBlock(block)

	out, err := driver.blockStorage.GetTransactionReceipt(&services.GetTransactionReceiptInput{
		Txhash:               []byte("will-not-be-found"),
		TransactionTimestamp: primitives.TimestampNano(time.Now().UnixNano()),
	})

	require.NoError(t, err, "transaction not found happy flow")
	require.Nil(t, out.TransactionReceipt, "represents an empty receipt")
	require.EqualValues(t, 1, out.BlockHeight, "last block height")
	require.EqualValues(t, block.ResultsBlock.Header.Timestamp(), out.BlockTimestamp, "last block timestamp")
}

// TODO return transaction receipt while the transaction timestamp is in the future (and too far ahead to be in the grace

func TestReturnTransactionReceipt(t *testing.T) {
	driver := NewDriver(t)
	driver.expectCommitStateDiff()

	block := builders.BlockPair().WithTransactions(10).WithReceiptsForTransactions().WithTimestampBloomFilter().WithTimestampNow().Build()
	driver.commitBlock(block)

	// it will be similar data transactions, but with different time stamps (and hashes..)
	block2 := builders.BlockPair().WithTransactions(10).WithReceiptsForTransactions().WithTimestampBloomFilter().WithTimestampNow().Build()
	driver.commitBlock(block2)

	// taking a transaction at 'random' (they were created at random)
	tx := block.TransactionsBlock.SignedTransactions[3].Transaction()
	txHash := digest.CalcTxHash(tx)

	// the block timestamp is just a couple of nanos ahead of the transactions, which is inside the grace
	out, err := driver.blockStorage.GetTransactionReceipt(&services.GetTransactionReceiptInput{
		Txhash:               txHash,
		TransactionTimestamp: tx.Timestamp(),
	})

	require.NoError(t, err, "receipt should be found in this flow")
	require.NotNil(t, out.TransactionReceipt, "receipt should be found in this flow")
	require.EqualValues(t, txHash, out.TransactionReceipt.Txhash(), "receipt should have the tx hash we looked for")
	require.EqualValues(t, 1, out.BlockHeight, "receipt should have the block height of the block containing the transaction")
	require.EqualValues(t, block.ResultsBlock.Header.Timestamp(), out.BlockTimestamp, "receipt should have the timestamp of the block containing the transaction")
}

// TODO return transaction receipt while the transaction timestamp is outside the grace (regular)
// TODO return transaction receipt while the transaction timestamp is at the expire window
// TODO return transaction receipt while the transaction timestamp is at the expire window and within the grace
