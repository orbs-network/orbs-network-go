package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReturnAllAvailableTransactionsFromTransactionPool(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		txCount := uint32(2)

		h.expectTxPoolToReturnXTransactions(txCount)

		txBlock, err := h.requestTransactionsBlock(ctx)
		if err != nil {
			t.Fatal("request transactions block failed:", err)
		}
		if uint32(len(txBlock.SignedTransactions)) != txCount {
			t.Fatalf("returned %d instead of %d", len(txBlock.SignedTransactions), txCount)
		}

		h.verifyTransactionsRequestedFromTransactionPool(t)
	})
}

// TODO v1 Decouple this test from TestReturnAllAvailableTransactionsFromTransactionPool()
// Presently if the latter fails, this test will fail too
func TestCreateBlock_HappyFlow(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		txCount := 2

		h.expectTxPoolToReturnXTransactions(uint32(txCount))
		h.expectStateHashToReturn([]byte{1, 2, 3, 4, 5})

		txBlock, err := h.requestTransactionsBlock(ctx)
		require.Nil(t, err, "request transactions block failed")
		h.expectVirtualMachineToReturnXTransactionReceipts(len(txBlock.SignedTransactions))
		rxBlock, err := h.requestResultsBlock(ctx, txBlock)
		require.Nil(t, err, "request results block failed")
		require.Equal(t, txCount, len(rxBlock.TransactionReceipts))
		h.verifyTransactionsRequestedFromTransactionPool(t)
	})
}
