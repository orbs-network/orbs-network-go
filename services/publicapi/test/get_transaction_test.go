package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGetTransactionStatus_GetCommitStatusFromTxPool(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newPublicApiHarness(ctx, 1*time.Second)

		txb := builders.Transaction().Builder()
		txHash := digest.CalcTxHash(txb.Build().Transaction())

		harness.transactionIsCommitedInPool()
		result, err := harness.papi.GetTransactionStatus(&services.GetTransactionStatusInput{
			ClientRequest: (&client.GetTransactionStatusRequestBuilder{
				Txhash: txHash,
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
		harness := newPublicApiHarness(ctx, 1*time.Second)

		txb := builders.Transaction().Builder()
		txHash := digest.CalcTxHash(txb.Build().Transaction())

		harness.transactionIsPendingInPool()
		result, err := harness.papi.GetTransactionStatus(&services.GetTransactionStatusInput{
			ClientRequest: (&client.GetTransactionStatusRequestBuilder{
				Txhash: txHash,
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
		harness := newPublicApiHarness(ctx, 1*time.Second)

		txb := builders.Transaction().Builder()
		txHash := digest.CalcTxHash(txb.Build().Transaction())

		harness.transactionIsNotInPoolIsInBlockStorage()
		result, err := harness.papi.GetTransactionStatus(&services.GetTransactionStatusInput{
			ClientRequest: (&client.GetTransactionStatusRequestBuilder{
				Txhash: txHash,
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
