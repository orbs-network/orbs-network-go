package publicapi

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
	"time"
)

func TestWaitForTransaction_ReturnsReceiptWhenCallbackArrivesAfterWaitIsCalled(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		w := newTxWaiter(ctx)

		receipt := builders.TransactionReceipt().WithRandomHash().Build()

		waitContext := w.createTxWaitCtx(receipt.Txhash())
		defer waitContext.cleanup()

		c := make(txResultChan)
		go func() {
			o, err := waitContext.until(100 * time.Millisecond)
			assert.NoError(t, err)
			if err != nil {
				close(c)
			} else {
				c <- o
			}
		}()

		w.reportCompleted(receipt, 0, 0)

		output := <-c
		require.NotZero(t, output)
		require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, output.TransactionStatus, "expected response with status committed")
	})
}

func TestWaitForTransaction_TwoWaitersOnSameTransactionDoNotBothBlock(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		w := newTxWaiter(ctx)

		receipt := builders.TransactionReceipt().WithRandomHash().Build()

		waitContext1 := w.createTxWaitCtx(receipt.Txhash())
		defer waitContext1.cleanup()
		waitContext2 := w.createTxWaitCtx(receipt.Txhash())
		defer waitContext2.cleanup()

		c := make(chan struct{})
		go func() {
			_, e1 := waitContext1.until(100 * time.Millisecond)
			_, e2 := waitContext2.until(100 * time.Millisecond)
			assert.Error(t, e1)
			assert.NoError(t, e2)
			close(c)
		}()

		w.reportCompleted(receipt, 0, 0)
		<-c
	})
}

func TestWaitForTransaction_Timeout(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		w := newTxWaiter(ctx)
		txHash := make(primitives.Sha256, 32)
		rand.Read(txHash)

		waitContext := w.createTxWaitCtx(txHash)
		defer waitContext.cleanup()

		_, err := waitContext.until(1 * time.Millisecond)

		require.EqualError(t, err, "timed out waiting for transaction result", "Timeout did not occur")
	})
}

func TestWaitForTransaction_GracefulShutdownFreesAllWaitingGoroutines(t *testing.T) {
	var w *txWaiter
	done := make(chan struct{})
	test.WithContext(func(ctx context.Context) {
		w = newTxWaiter(ctx)

		var waitTillCancelled = func() {
			txHash := make(primitives.Sha256, 32)
			rand.Read(txHash)
			waitContext := w.createTxWaitCtx(txHash)
			defer waitContext.cleanup()

			startTime := time.Now()
			_, err := waitContext.until(1 * time.Second)
			assert.Error(t, err, "expected waiting to be aborted")
			assert.WithinDuration(t, time.Now(), startTime, 100*time.Millisecond, "expected not to reach timeout")
			done <- struct{}{}
		}
		go waitTillCancelled()
		go waitTillCancelled()
	})
	<-done
	<-done
}
