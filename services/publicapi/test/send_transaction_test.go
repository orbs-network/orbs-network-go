// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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
		harness := newPublicApiHarness(ctx, t, time.Millisecond, time.Minute)
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
		harness := newPublicApiHarness(ctx, t, time.Second, time.Minute)

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
		harness := newPublicApiHarness(ctx, t, time.Second, time.Minute)

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
		harness := newPublicApiHarness(ctx, t, txTimeout, time.Minute)

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
		require.WithinDuration(t, time.Now(), start, txTimeout+100*time.Millisecond, "timeout duration seems much longer than expected")
		require.NotNil(t, result, "Send transaction returned nil instead of object")
	})
}

func TestSendTransactionAsync_ReturnsImmediately(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		txTimeout := 100 * time.Hour // infinity - won't actually wait please don't change
		harness := newPublicApiHarness(ctx, t, txTimeout, time.Minute)

		txb := builders.Transaction().Builder()
		harness.onAddNewTransaction(func() {})

		start := time.Now()
		result, _ := harness.papi.SendTransactionAsync(ctx, &services.SendTransactionInput{
			ClientRequest: (&client.SendTransactionRequestBuilder{
				SignedTransaction: txb,
			}).Build(),
		})

		harness.verifyMocks(t) // contract test

		// value test
		require.WithinDuration(t, time.Now(), start, 10*time.Second, "timeout duration exceeded")
		require.EqualValues(t, protocol.TRANSACTION_STATUS_PENDING, result.ClientResponse.TransactionStatus(), "should be pending")
	})
}
