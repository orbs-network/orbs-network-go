package test

import (
	"testing"
)

func TestReturnAllAvailableTransactionsFromTransactionPool(t *testing.T) {

	h := newHarness()
	txCount := h.config.MinimumTransactionsInBlock() + 1

	h.expectTransactionsRequestedFromTransactionPool(txCount)

	txBlock, err := h.requestTransactionsBlock()
	if err != nil {
		t.Fatal("request transactions block failed:", err)
	}
	if len(txBlock.SignedTransactions) != txCount {
		t.Fatalf("returned %d instead of %d", len(txBlock.SignedTransactions), txCount)
	}

	h.verifyTransactionsRequestedFromTransactionPool(t)
}

func TestRetryWhenNotEnoughTransactionsPendingOnTransactionPool(t *testing.T) {

	h := newHarness()

	if h.config.MinimumTransactionsInBlock() <= 0 {
		t.Errorf("must set MinimumTransactionsInBlock > 0 in test config")
	}

	txCount := h.config.MinimumTransactionsInBlock() - 1


	// TODO: The order of expect() is reversed: Tal should fix it and the order of expects() here should then be reversed!!!
	//h.expectTransactionsNoLongerRequestedFromTransactionPool()
	h.expectTransactionsRequestedFromTransactionPool(txCount)
	h.expectTransactionsRequestedFromTransactionPool(0)


	txBlock, err := h.requestTransactionsBlock()
	if err != nil {
		t.Fatal("request transactions block failed:", err)
	}

	if len(txBlock.SignedTransactions) != txCount {
		t.Fatalf("returned %d instead of %d", len(txBlock.SignedTransactions), txCount)
	}

	h.verifyTransactionsRequestedFromTransactionPool(t)
}

