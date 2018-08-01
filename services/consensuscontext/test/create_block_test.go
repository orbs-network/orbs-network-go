package test

import (
	"testing"
)

func TestReturnAllAvailableTransactionsFromTransactionPool(t *testing.T) {

	h := newHarness()
	txCount := 3

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
	txCount := 3


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

