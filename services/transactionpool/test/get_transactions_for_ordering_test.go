// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGetTransactionsForOrderingAsOfFutureBlockHeightTimesOutWhenNoBlockIsCommitted(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, parent *test.ConcurrencyHarness) {
		h := newHarness(parent).start(ctx)

		_, err := h.txpool.GetTransactionsForOrdering(ctx, &services.GetTransactionsForOrderingInput{
			CurrentBlockHeight:      3,
			PrevBlockTimestamp:      0,
			MaxNumberOfTransactions: 1,
		})

		require.EqualError(t, errors.Cause(err), "context deadline exceeded", "did not time out")
	})
}

func TestGetTransactionsForOrderingAsOfFutureBlockHeightTimesOutWhenContextIsCancelled(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, parent *test.ConcurrencyHarness) {
		h := newHarness(parent).start(ctx)

		// init a cancelled child context for the exercise
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel()

		_, err := h.txpool.GetTransactionsForOrdering(cancelledCtx, &services.GetTransactionsForOrderingInput{
			CurrentBlockHeight:      3,
			PrevBlockTimestamp:      0,
			MaxNumberOfTransactions: 1,
		})

		require.EqualError(t, errors.Cause(err), context.Canceled.Error(), "when presented with a cancelled context getTransactionsForOrdering did not cancel immediately")
	})
}

func TestGetTransactionsForOrderingAsOfFutureBlockHeightResolvesOutWhenBlockIsCommitted(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, parent *test.ConcurrencyHarness) {
		h := newHarness(parent).start(ctx)

		h.assumeBlockStorageAtHeight(1)
		h.ignoringTransactionResults()
		h.reportTransactionsAsCommitted(ctx)

		doneWait := make(chan error)
		go func() {
			_, err := h.txpool.GetTransactionsForOrdering(ctx, &services.GetTransactionsForOrderingInput{
				CurrentBlockHeight:      3,
				PrevBlockTimestamp:      0,
				MaxNumberOfTransactions: 1,
			})
			doneWait <- err
		}()

		require.NoError(t, <-doneWait, "did not resolve after block has been committed")
	})
}

func TestGetTransactionsForOrderingWaitsForAdditionalTransactionsIfUnderMinimum(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, parent *test.ConcurrencyHarness) {
		h := newHarnessWithInfiniteTimeBetweenEmptyBlocks(parent).start(ctx)

		ch := make(chan *services.GetTransactionsForOrderingOutput)

		go func() {
			out, err := h.getTransactionsForOrdering(ctx, 2, 1)
			require.NoError(t, err)
			ch <- out
		}()

		time.Sleep(50 * time.Millisecond) // make sure we wait, also deals with https://github.com/orbs-network/orbs-network-go/issues/852
		h.handleForwardFrom(ctx, otherNodeKeyPair, builders.TransferTransaction().Build())

		out := <-ch
		require.EqualValues(t, 1, len(out.SignedTransactions), "did not wait for transaction to reach pool")
		require.NotZero(t, out.ProposedBlockTimestamp, "proposed block timestamp is zero")
	})
}

func TestGetTransactionsForOrderingDoesNotWaitForAdditionalTransactionsIfContextIsCancelled(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, parent *test.ConcurrencyHarness) {
		h := newHarnessWithInfiniteTimeBetweenEmptyBlocks(parent).start(ctx)

		// init a cancelled child context for the exercise
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel()

		out, err := h.getTransactionsForOrdering(cancelledCtx, 2, 1)

		require.EqualValues(t, 0, len(out.SignedTransactions), "when presented with a cancelled context, and not enough transactions in pool, getTransactionsForOrdering did not return an empty block immediately")
		require.NoError(t, err, "when presented with a cancelled context, and not enough transactions in pool, getTransactionsForOrdering should return gracefully")
	})
}

func TestGetTransactionsForOrderingOnGenesisBlockReturnsZeroTransactions(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, parent *test.ConcurrencyHarness) {
		h := newHarness(parent).start(ctx)
		h.handleForwardFrom(ctx, otherNodeKeyPair, builders.TransferTransaction().Build())

		out, err := h.txpool.GetTransactionsForOrdering(ctx, &services.GetTransactionsForOrderingInput{
			CurrentBlockHeight:      1,
			PrevBlockTimestamp:      0,
			MaxNumberOfTransactions: 1,
		})

		require.NoError(t, err, "GetTransactionsForOrdering should not fail")
		require.Zero(t, len(out.SignedTransactions), "number of transactions should be zero")
		require.NotZero(t, out.ProposedBlockTimestamp, "proposed block timestamp should not be zero")
	})
}

func TestGetTransactionsForOrderingAfterGenesisBlockReturnsNonZeroTransactions(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, parent *test.ConcurrencyHarness) {
		h := newHarness(parent).start(ctx)
		h.handleForwardFrom(ctx, otherNodeKeyPair, builders.TransferTransaction().Build())

		out, err := h.txpool.GetTransactionsForOrdering(ctx, &services.GetTransactionsForOrderingInput{
			CurrentBlockHeight:      2,
			PrevBlockTimestamp:      0,
			MaxNumberOfTransactions: 1,
		})

		require.NoError(t, err, "GetTransactionsForOrdering should not fail")
		require.NotZero(t, len(out.SignedTransactions), "number of transactions should not be zero")
		require.NotZero(t, out.ProposedBlockTimestamp, "proposed block timestamp should not be zero")
	})
}
