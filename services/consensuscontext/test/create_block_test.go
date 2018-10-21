package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReturnAllAvailableTransactionsFromTransactionPool(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness()
		txCount := h.config.ConsensusContextMinimumTransactionsInBlock() + 1

		h.expectTransactionsRequestedFromTransactionPool(txCount)

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

func TestRetryWhenNotEnoughTransactionsPendingOnTransactionPool(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness()

		if h.config.ConsensusContextMinimumTransactionsInBlock() <= 1 {
			t.Errorf("must set ConsensusContextMinimumTransactionsInBlock > 1 in test config, now it is %v", h.config.ConsensusContextMinimumTransactionsInBlock())
		}

		txCount := h.config.ConsensusContextMinimumTransactionsInBlock() - 1

		h.expectTransactionsRequestedFromTransactionPool(0)
		h.expectTransactionsRequestedFromTransactionPool(txCount)

		txBlock, err := h.requestTransactionsBlock(ctx)
		require.NoError(t, err, "request transactions block failed:", err)

		if uint32(len(txBlock.SignedTransactions)) != txCount {
			t.Fatalf("returned %d instead of %d", len(txBlock.SignedTransactions), txCount)
		}

		h.verifyTransactionsRequestedFromTransactionPool(t)
	})
}
