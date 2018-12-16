package publicapi

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestPublicApiGetTxProof_PrepareResponse(t *testing.T) {
	blockTime := primitives.TimestampNano(time.Now().Nanosecond())
	status := &services.GetTransactionStatusOutput{
		ClientResponse: (&client.GetTransactionStatusResponseBuilder{
			RequestStatus:      protocol.REQUEST_STATUS_COMPLETED,
			TransactionReceipt: nil, // doesn't matter here
			TransactionStatus:  protocol.TRANSACTION_STATUS_COMMITTED,
			BlockHeight:        126,
			BlockTimestamp:     blockTime,
		}).Build(),
	}

	// TODO issue 67 raw data builder ?
	proof := &services.GenerateReceiptProofOutput{
		Proof: (&protocol.ReceiptProofBuilder{
			Header:       nil,
			BlockProof:   nil,
			ReceiptProof: nil,
		}).Build(),
	}

	response := toGetReceiptOutput(status, proof)

	// TODO issue 67 raw data
	//test.RequireCmpEqual(t, receipt, response.ClientResponse.Proof(), "Transaction receipt is not equal")
	require.EqualValues(t, 126, response.ClientResponse.BlockHeight(), "Block height response is wrong")
	require.EqualValues(t, blockTime, response.ClientResponse.BlockTimestamp(), "Block time response is wrong")
	require.EqualValues(t, protocol.TRANSACTION_STATUS_COMMITTED, response.ClientResponse.TransactionStatus(), "status response is wrong")
}

func TestPublicApiGetTxProof_PrepareResponseNilProof(t *testing.T) {
	blockTime := primitives.TimestampNano(time.Now().Nanosecond())

	response := toGetReceiptOutput(&services.GetTransactionStatusOutput{
		ClientResponse: (&client.GetTransactionStatusResponseBuilder{
			RequestStatus:      protocol.REQUEST_STATUS_IN_PROCESS,
			TransactionReceipt: nil,
			TransactionStatus:  protocol.TRANSACTION_STATUS_PENDING,
			BlockHeight:        8,
			BlockTimestamp:     blockTime,
		}).Build(),
	}, nil)

	require.EqualValues(t, []byte{}, response.ClientResponse.Proof(), "Transaction receipt is not equal")
	require.Equal(t, 0, len(response.ClientResponse.Proof()), "Transaction receipt is not equal") // different way
	require.EqualValues(t, 8, response.ClientResponse.BlockHeight(), "Block height response is wrong")
	require.EqualValues(t, blockTime, response.ClientResponse.BlockTimestamp(), "Block time response is wrong")
	require.EqualValues(t, protocol.TRANSACTION_STATUS_PENDING, response.ClientResponse.TransactionStatus(), "status response is wrong")
}

func TestPublicApiGetTxProof_EmptyResponse(t *testing.T) {
	response := toGetReceiptOutput(nil, nil)

	require.EqualValues(t, []byte{}, response.ClientResponse.Proof(), "Transaction proof is not equal")
	require.Equal(t, 0, len(response.ClientResponse.Proof()), "Transaction proof is not equal") // different way
	require.EqualValues(t, 0, response.ClientResponse.BlockHeight(), "Block height response is wrong")
	require.EqualValues(t, 0, response.ClientResponse.BlockTimestamp(), "Block time response is wrong")
	require.EqualValues(t, protocol.TRANSACTION_STATUS_NO_RECORD_FOUND, response.ClientResponse.TransactionStatus(), "status response is wrong")
}
