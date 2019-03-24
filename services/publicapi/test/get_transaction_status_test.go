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
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGetTransactionStatus_GetCommittedStatusFromTxPool(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newPublicApiHarness(ctx, t, time.Second, time.Minute)

		harness.transactionIsCommittedInPool()
		result, err := harness.papi.GetTransactionStatus(ctx, &services.GetTransactionStatusInput{
			ClientRequest: (&client.GetTransactionStatusRequestBuilder{
				TransactionRef: builders.TransactionRef().Builder(),
			}).Build(),
		})

		harness.verifyMocks(t) // contract test

		// value test
		require.NoError(t, err, "error happened when it should not")
		require.NotNil(t, result, "get transaction status returned nil instead of object")
		require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, result.ClientResponse.TransactionStatus(), "got wrong status")
		require.NotNil(t, result.ClientResponse.TransactionReceipt(), "got empty receipt")
	})
}

func TestGetTransactionStatus_GetPendingStatusFromTxPool(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newPublicApiHarness(ctx, t, time.Second, time.Minute)

		harness.transactionIsPendingInPool()
		result, err := harness.papi.GetTransactionStatus(ctx, &services.GetTransactionStatusInput{
			ClientRequest: (&client.GetTransactionStatusRequestBuilder{
				TransactionRef: builders.TransactionRef().Builder(),
			}).Build(),
		})

		harness.verifyMocks(t) // contract test

		// value test
		require.NoError(t, err, "error happened when it should not")
		require.NotNil(t, result, "get transaction status returned nil instead of object")
		require.Equal(t, protocol.TRANSACTION_STATUS_PENDING, result.ClientResponse.TransactionStatus(), "got wrong status")
		test.RequireCmpEqual(t, (*protocol.TransactionReceiptBuilder)(nil).Build(), result.ClientResponse.TransactionReceipt(), "Transaction receipt is not equal")
	})
}

func TestGetTransactionStatus_GetTxFromBlockStorage(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newPublicApiHarness(ctx, t, time.Second, time.Minute)

		harness.transactionIsNotInPoolIsInBlockStorage()
		result, err := harness.papi.GetTransactionStatus(ctx, &services.GetTransactionStatusInput{
			ClientRequest: (&client.GetTransactionStatusRequestBuilder{
				TransactionRef: builders.TransactionRef().Builder(),
			}).Build(),
		})

		harness.verifyMocks(t) // contract test

		// value test
		require.NoError(t, err, "error happened when it should not")
		require.NotNil(t, result, "get transaction status returned nil instead of object")
		require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, result.ClientResponse.TransactionStatus(), "got wrong status")
		require.NotNil(t, result.ClientResponse.TransactionReceipt(), "got empty receipt")
	})
}
