package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetTransactionsForOrderingAsOfFutureBlockHeightTimesOutWhenNoBlockIsCommitted(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(ctx)

		_, err := h.txpool.GetTransactionsForOrdering(ctx, &services.GetTransactionsForOrderingInput{
			CurrentBlockHeight:      3,
			CurrentBlockTimestamp:   0,
			MaxNumberOfTransactions: 1,
		})

		require.EqualError(t, errors.Cause(err), "context deadline exceeded", "did not time out")
	})
}

func TestGetTransactionsForOrderingAsOfFutureBlockHeightResolvesOutWhenBlockIsCommitted(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(ctx)

		h.assumeBlockStorageAtHeight(1)
		h.ignoringTransactionResults()
		h.reportTransactionsAsCommitted(ctx)

		doneWait := make(chan error)
		go func() {
			_, err := h.txpool.GetTransactionsForOrdering(ctx, &services.GetTransactionsForOrderingInput{
				CurrentBlockHeight:      3,
				CurrentBlockTimestamp:   0,
				MaxNumberOfTransactions: 1,
			})
			doneWait <- err
		}()

		require.NoError(t, <-doneWait, "did not resolve after block has been committed")
	})
}

func TestGetTransactionsForOrderingWaitsForAdditionalTransactionsIfUnderMinimum(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(ctx)

		ch := make(chan int)

		go func() {
			out, err := h.getTransactionsForOrdering(ctx, 2, 1)
			require.NoError(t, err)
			ch <- len(out.SignedTransactions)
		}()

		h.handleForwardFrom(ctx, otherNodeKeyPair, builders.TransferTransaction().Build())

		select {
		case numOfTxs := <-ch: // required so that if the require.NoError in the goroutine fails, we don't wait on reading from a channel that will never be written to
			require.EqualValues(t, 1, numOfTxs, "did not wait for transaction to reach pool")
		case <-ctx.Done():
		}
	})
}
