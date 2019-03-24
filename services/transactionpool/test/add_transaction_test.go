// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/services/transactionpool"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestForwardsANewValidTransactionUsingGossip(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t).start(ctx)

		tx := builders.TransferTransaction().Build()

		hash, _, _ := transactionpool.HashTransactions(tx)
		sig, _ := digest.SignAsNode(thisNodeKeyPair.PrivateKey(), hash)

		h.expectTransactionsToBeForwarded(sig, tx)

		_, err := h.addNewTransaction(ctx, tx)
		require.NoError(t, err, "a valid transaction was not added to pool")
		require.NoError(t, test.EventuallyVerify(h.config.TransactionPoolPropagationBatchingTimeout()*10, h.gossip), "mocks were not called as expected")
	})
}

func TestDoesNotForwardInvalidTransactionsUsingGossip(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t).start(ctx)

		tx := builders.TransferTransaction().WithTimestampInFarFuture().Build()
		h.expectNoTransactionsToBeForwarded()

		_, err := h.addNewTransaction(ctx, tx)

		require.Error(t, err, "an invalid transaction was added to the pool")
		require.NoError(t, test.ConsistentlyVerify(10*time.Millisecond, h.gossip), "mocks were not called as expected")
	})
}

func TestDoesNotAddTransactionsThatFailedPreOrderChecks_GlobalPreOrder(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t).allowingErrorsMatching("error validating transaction for preorder").start(ctx)
		tx := builders.TransferTransaction().Build()
		h.failPreOrderCheckFor(func(t *protocol.SignedTransaction) bool {
			return t == tx
		}, protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER)

		h.ignoringForwardMessages()

		blockHeight := primitives.BlockHeight(3)
		blockTime := primitives.TimestampNano(time.Now().UnixNano())
		h.fastForwardToHeightAndTime(ctx, blockHeight, blockTime)

		out, err := h.addNewTransaction(ctx, tx)

		require.NotNil(t, out, "output must not be nil even on errors")
		require.Equal(t, blockHeight, out.BlockHeight)
		require.Equal(t, blockTime, out.BlockTimestamp)

		require.Error(t, err, "an transaction that failed pre-order checks was added to the pool")
		require.IsType(t, &transactionpool.ErrTransactionRejected{}, err, "error was not of the expected type")

		typedError := err.(*transactionpool.ErrTransactionRejected)
		require.Equal(t, protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER, typedError.TransactionStatus, "error did not contain expected transaction status")

		require.NoError(t, h.verifyMocks(), "mocks were not called as expected")
	})
}

func TestDoesNotAddTransactionsThatFailedPreOrderChecks_SignatureMismatch(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t).allowingErrorsMatching("error validating transaction for preorder").start(ctx)
		tx := builders.TransferTransaction().Build()
		h.failPreOrderCheckFor(func(t *protocol.SignedTransaction) bool {
			return t == tx
		}, protocol.TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH)

		h.ignoringForwardMessages()

		blockHeight := primitives.BlockHeight(3)
		blockTime := primitives.TimestampNano(time.Now().UnixNano())
		h.fastForwardToHeightAndTime(ctx, blockHeight, blockTime)

		out, err := h.addNewTransaction(ctx, tx)

		require.NotNil(t, out, "output must not be nil even on errors")
		require.Equal(t, blockHeight, out.BlockHeight)
		require.Equal(t, blockTime, out.BlockTimestamp)

		require.Error(t, err, "an transaction that failed pre-order checks was added to the pool")
		require.IsType(t, &transactionpool.ErrTransactionRejected{}, err, "error was not of the expected type")

		typedError := err.(*transactionpool.ErrTransactionRejected)
		require.Equal(t, protocol.TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH, typedError.TransactionStatus, "error did not contain expected transaction status")

		require.NoError(t, h.verifyMocks(), "mocks were not called as expected")
	})
}

func TestDoesNotAddTheSameTransactionTwice(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t).allowingErrorsMatching("error adding transaction to pending pool").start(ctx)

		tx := builders.TransferTransaction().Build()
		h.ignoringForwardMessages()

		h.addNewTransaction(ctx, tx)

		receipt, err := h.addNewTransaction(ctx, tx)

		require.Error(t, err, "a transaction was added twice to the pool")
		require.NotNil(t, receipt, "receipt should never be nil")
		require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING, receipt.TransactionStatus, "expected transaction status: duplicate pending")
		require.NoError(t, h.verifyMocks(), "mocks were not called as expected")
	})
}

func TestReturnsReceiptForTransactionThatHasAlreadyBeenCommitted(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t).start(ctx)

		tx := builders.TransferTransaction().Build()
		h.ignoringForwardMessages()
		h.ignoringTransactionResults()

		h.addNewTransaction(ctx, tx)
		_, err := h.reportTransactionsAsCommitted(ctx, tx)
		require.NoError(t, err, "committing a transaction returned an unexpected error")

		receipt, err := h.addNewTransaction(ctx, tx)

		require.NoError(t, err, "a committed transaction that was added again was wrongly rejected")
		require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, receipt.TransactionStatus, "expected transaction status to be committed")
		require.Equal(t, digest.CalcTxHash(tx.Transaction()), receipt.TransactionReceipt.Txhash(), "expected transaction receipt to contain transaction hash")

		require.NoError(t, h.verifyMocks(), "mocks were not called as expected")
	})
}

func TestDoesNotAddTransactionIfPoolIsFull(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarnessWithSizeLimit(t, 1).allowingErrorsMatching("error adding transaction to pending pool").start(ctx)

		h.expectNoTransactionsToBeForwarded()

		tx := builders.TransferTransaction().Build()
		receipt, err := h.addNewTransaction(ctx, tx)

		require.Error(t, err, "a transaction was added to a full pool")
		require.NotNil(t, receipt, "receipt should never be nil")
		require.Equal(t, protocol.TRANSACTION_STATUS_REJECTED_CONGESTION, receipt.TransactionStatus, "expected transaction status: congestion")
		require.NoError(t, h.verifyMocks(), "mocks were not called as expected")
	})
}
