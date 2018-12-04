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

func TestGetTransactionReceipt_GetCommitStatusFromTxPool(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newPublicApiHarness(ctx, 1*time.Second)

		txb := builders.Transaction().Builder()
		txHash := digest.CalcTxHash(txb.Build().Transaction())

		harness.transactionHasProof()
		result, err := harness.papi.GetTransactionReceiptProof(ctx, &services.GetTransactionReceiptProofInput{
			ClientRequest: (&client.GetTransactionReceiptProofRequestBuilder{
				ProtocolVersion:      txb.Transaction.ProtocolVersion,
				VirtualChainId:       txb.Transaction.VirtualChainId,
				TransactionTimestamp: txb.Transaction.Timestamp,
				Txhash:               txHash,
				BlockHeight:          0, // TODO ?
			}).Build(),
		})

		harness.verifyMocks(t) // contract test

		// value test
		require.NoError(t, err, "error happened when it should not")
		require.NotNil(t, result, "get transaction receipt returned nil instead of object")
		require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, result.ClientResponse.TransactionStatus(), "got wrong status")
		require.NotNil(t, result.ClientResponse.Proof(), "got empty receipt proof")
	})
}

func TestGetTransactionReceipt_GetPendingStatusFromTxPool(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newPublicApiHarness(ctx, 1*time.Second)

		txb := builders.Transaction().Builder()
		txHash := digest.CalcTxHash(txb.Build().Transaction())

		harness.transactionPendingNoProofCalled()
		result, err := harness.papi.GetTransactionReceiptProof(ctx, &services.GetTransactionReceiptProofInput{
			ClientRequest: (&client.GetTransactionReceiptProofRequestBuilder{
				ProtocolVersion:      txb.Transaction.ProtocolVersion,
				VirtualChainId:       txb.Transaction.VirtualChainId,
				TransactionTimestamp: txb.Transaction.Timestamp,
				Txhash:               txHash,
				BlockHeight:          0, // TODO ?
			}).Build(),
		})

		harness.verifyMocks(t) // contract test

		// value test
		require.NoError(t, err, "error happened when it should not")
		require.NotNil(t, result, "get transaction receipt returned nil instead of object")
		require.Equal(t, protocol.TRANSACTION_STATUS_PENDING, result.ClientResponse.TransactionStatus(), "got wrong status")
		require.Equal(t, 0, len(result.ClientResponse.Proof().Raw()), "Transaction proof is not equal")
	})
}

func TestGetTransactionReceipt_NoRecordsFound(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newPublicApiHarness(ctx, 1*time.Second)

		txb := builders.Transaction().Builder()
		txHash := digest.CalcTxHash(txb.Build().Transaction())

		harness.getTransactionStatusFailed()
		result, err := harness.papi.GetTransactionReceiptProof(ctx, &services.GetTransactionReceiptProofInput{
			ClientRequest: (&client.GetTransactionReceiptProofRequestBuilder{
				ProtocolVersion:      txb.Transaction.ProtocolVersion,
				VirtualChainId:       txb.Transaction.VirtualChainId,
				TransactionTimestamp: txb.Transaction.Timestamp,
				Txhash:               txHash,
				BlockHeight:          0, // TODO ?
			}).Build(),
		})

		harness.verifyMocks(t) // contract test

		// value test
		require.Error(t, err, "error did not happen when it should")
		require.NotNil(t, result, "get transaction receipt returned nil instead of object")
		require.Equal(t, protocol.TRANSACTION_STATUS_NO_RECORD_FOUND, result.ClientResponse.TransactionStatus(), "got wrong status")
		require.Equal(t, 0, len(result.ClientResponse.Proof().Raw()), "Transaction proof is not equal")
	})
}
