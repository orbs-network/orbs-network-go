package test

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSendTransaction_AlreadyCommitted(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newPublicApiHarness(ctx, time.Millisecond, time.Minute)
		harness.addTransactionReturnsAlreadyCommitted()

		result, err := harness.papi.SendTransaction(ctx, &services.SendTransactionInput{
			ClientRequest: (&client.SendTransactionRequestBuilder{
				SignedTransaction: builders.Transaction().Builder()}).Build(),
		})

		harness.verifyMocks(t) // contract test

		// value test
		require.NoError(t, err, "error happened when it should not")
		require.NotNil(t, result, "Send transaction returned nil instead of object")
	})
}

func TestSendTransaction_BlocksUntilTransactionCompletes(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newPublicApiHarness(ctx, time.Second, time.Minute)

		txb := builders.Transaction().Builder()
		harness.onAddNewTransaction(func() {
			harness.papi.HandleTransactionResults(ctx, &handlers.HandleTransactionResultsInput{
				TransactionReceipts: []*protocol.TransactionReceipt{builders.TransactionReceipt().WithTransaction(txb.Build().Transaction()).Build()},
			})
		})

		result, err := harness.papi.SendTransaction(ctx, &services.SendTransactionInput{
			ClientRequest: (&client.SendTransactionRequestBuilder{
				SignedTransaction: txb,
			}).Build(),
		})

		harness.verifyMocks(t) // contract test

		require.NoError(t, err, "error happened when it should not")
		require.NotNil(t, result, "Send transaction returned nil instead of object")
	})
}

func TestSendTransaction_BlocksUntilTransactionErrors(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newPublicApiHarness(ctx, time.Second, time.Minute)

		txb := builders.Transaction().Builder()
		txHash := digest.CalcTxHash(txb.Build().Transaction())

		harness.onAddNewTransaction(func() {
			harness.papi.HandleTransactionError(ctx, &handlers.HandleTransactionErrorInput{
				Txhash:            txHash,
				TransactionStatus: protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED,
			})
		})

		result, err := harness.papi.SendTransaction(ctx, &services.SendTransactionInput{
			ClientRequest: (&client.SendTransactionRequestBuilder{
				SignedTransaction: txb,
			}).Build(),
		})

		harness.verifyMocks(t) // contract test

		require.NoError(t, err, "error happened when it should not")
		require.NotNil(t, result, "Send transaction returned nil instead of object")
		require.EqualValues(t, protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED, result.ClientResponse.TransactionStatus(), "got wrong status")
	})
}

func TestSendTransaction_TimesOut(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		txTimeout := 10 * time.Millisecond
		harness := newPublicApiHarness(ctx, txTimeout, time.Minute)

		txb := builders.Transaction().Builder()
		harness.onAddNewTransaction(func() {})

		start := time.Now()
		result, err := harness.papi.SendTransaction(ctx, &services.SendTransactionInput{
			ClientRequest: (&client.SendTransactionRequestBuilder{
				SignedTransaction: txb,
			}).Build(),
		})

		harness.verifyMocks(t) // contract test

		// value test
		txHash := digest.CalcTxHash(txb.Build().Transaction())

		require.Contains(t, err.Error(), fmt.Sprintf("waiting aborted due to context termination for key %s", txHash.String()))
		require.WithinDuration(t, time.Now(), start, 2*txTimeout, "timeout duration exceeded")
		require.NotNil(t, result, "Send transaction returned nil instead of object")
	})
}

func TestSendTransaction_ReturnImmediately(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		txTimeout := 100 * time.Second // won't actually wait please don't change
		harness := newPublicApiHarness(ctx, txTimeout, time.Minute)

		txb := builders.Transaction().Builder()
		harness.onAddNewTransaction(func() {})

		start := time.Now()
		result, _ := harness.papi.SendTransaction(ctx, &services.SendTransactionInput{
			ClientRequest: (&client.SendTransactionRequestBuilder{
				SignedTransaction: txb,
			}).Build(),
			ReturnImmediately: 1,
		})

		harness.verifyMocks(t) // contract test

		// value test
		require.WithinDuration(t, time.Now(), start, 1*time.Millisecond, "timeout duration exceeded")
		require.EqualValues(t, protocol.TRANSACTION_STATUS_PENDING, result.ClientResponse.TransactionStatus(), "should be pending")
	})
}
