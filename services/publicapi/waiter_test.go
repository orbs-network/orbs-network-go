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
		w := newWaiter(ctx)

		receipt := builders.TransactionReceipt().WithRandomHash().Build()

		wait := w.prepareFor(receipt.Txhash())
		defer wait.cleanup()

		c := make(txResultChan)
		go func() {
			o, err := wait.until(100 * time.Millisecond)
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
		w := newWaiter(ctx)

		receipt := builders.TransactionReceipt().WithRandomHash().Build()

		wait := w.prepareFor(receipt.Txhash())
		defer wait.cleanup()
		wait2 := w.prepareFor(receipt.Txhash())
		defer wait2.cleanup()

		c := make(chan struct{})
		go func() {
			_, e1 := wait.until(100 * time.Millisecond)
			_, e2 := wait2.until(100 * time.Millisecond)
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
		w := newWaiter(ctx)
		var txHash primitives.Sha256
		rand.Read(txHash)

		wait := w.prepareFor(txHash)
		defer wait.cleanup()

		_, err := wait.until(1 * time.Millisecond)

		require.EqualError(t, err, "timed out waiting for transaction result", "Timeout did not occur")
	})
}
