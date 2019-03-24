// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package publicapi

import (
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGetTransactionReceiptProof_PrepareResponse(t *testing.T) {
	ctrlRand := rand.NewControlledRand(t)
	blockTime := primitives.TimestampNano(time.Now().UnixNano())
	receipt := builders.TransactionReceipt().WithRandomHash(ctrlRand).Builder()

	txStatusOutput := &services.GetTransactionStatusOutput{
		ClientResponse: (&client.GetTransactionStatusResponseBuilder{
			RequestResult: &client.RequestResultBuilder{
				RequestStatus:  protocol.REQUEST_STATUS_COMPLETED,
				BlockHeight:    126,
				BlockTimestamp: blockTime,
			},
			TransactionStatus:  protocol.TRANSACTION_STATUS_COMMITTED,
			TransactionReceipt: receipt,
		}).Build(),
	}

	proof := (&protocol.ReceiptProofBuilder{
		Header:       &protocol.ResultsBlockHeaderBuilder{},
		BlockProof:   &protocol.ResultsBlockProofBuilder{},
		ReceiptProof: primitives.MerkleTreeProof([]byte{0x01, 0x02}),
	}).Build()
	proofOutput := &services.GenerateReceiptProofOutput{
		Proof: proof,
	}

	response := toGetTxProofOutput(txStatusOutput, proofOutput)

	require.EqualValues(t, protocol.REQUEST_STATUS_COMPLETED, response.ClientResponse.RequestResult().RequestStatus(), "Request status is wrong")
	require.EqualValues(t, 126, response.ClientResponse.RequestResult().BlockHeight(), "Block height response is wrong")
	require.EqualValues(t, blockTime, response.ClientResponse.RequestResult().BlockTimestamp(), "Block time response is wrong")

	require.EqualValues(t, protocol.TRANSACTION_STATUS_COMMITTED, response.ClientResponse.TransactionStatus(), "txStatus response is wrong")
	test.RequireCmpEqual(t, receipt.Build(), response.ClientResponse.TransactionReceipt(), "Transaction receipt is not equal")

	require.EqualValues(t, proof.Raw(), response.ClientResponse.PackedProof(), "Packed proof is not equal")
}

func TestGetTransactionReceiptProof_PrepareResponse_NilProof(t *testing.T) {
	blockTime := primitives.TimestampNano(time.Now().UnixNano())

	txStatusOutput := &services.GetTransactionStatusOutput{
		ClientResponse: (&client.GetTransactionStatusResponseBuilder{
			RequestResult: &client.RequestResultBuilder{
				RequestStatus:  protocol.REQUEST_STATUS_IN_PROCESS,
				BlockHeight:    8,
				BlockTimestamp: blockTime,
			},
			TransactionStatus:  protocol.TRANSACTION_STATUS_PENDING,
			TransactionReceipt: nil,
		}).Build(),
	}

	response := toGetTxProofOutput(txStatusOutput, nil)

	require.EqualValues(t, protocol.REQUEST_STATUS_IN_PROCESS, response.ClientResponse.RequestResult().RequestStatus(), "Request status is wrong")
	require.EqualValues(t, 8, response.ClientResponse.RequestResult().BlockHeight(), "Block height response is wrong")
	require.EqualValues(t, blockTime, response.ClientResponse.RequestResult().BlockTimestamp(), "Block time response is wrong")

	require.EqualValues(t, protocol.TRANSACTION_STATUS_PENDING, response.ClientResponse.TransactionStatus(), "txStatus response is wrong")
	require.Equal(t, 0, len(response.ClientResponse.TransactionReceipt().Raw()), "Transaction receipt is not equal")
	test.RequireCmpEqual(t, (*protocol.TransactionReceiptBuilder)(nil).Build(), response.ClientResponse.TransactionReceipt(), "Transaction receipt is not equal")

	require.Equal(t, 0, len(response.ClientResponse.PackedProof()), "Transaction proof is not equal")
	require.EqualValues(t, []byte{}, response.ClientResponse.PackedProof(), "Transaction proof is not equal")
}

func TestGetTransactionReceiptProof_EmptyResponse(t *testing.T) {
	response := toGetTxProofOutput(nil, nil)

	require.EqualValues(t, protocol.REQUEST_STATUS_NOT_FOUND, response.ClientResponse.RequestResult().RequestStatus(), "Request status is wrong")
	require.EqualValues(t, protocol.TRANSACTION_STATUS_NO_RECORD_FOUND, response.ClientResponse.TransactionStatus(), "txStatus response is wrong")
}
