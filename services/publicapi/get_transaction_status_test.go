// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package publicapi

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGetTransactionStatus_PrepareResponse(t *testing.T) {
	ctrlRand := rand.NewControlledRand(t)
	cfg := config.ForPublicApiTests(22, 0, time.Minute)
	blockTime := primitives.TimestampNano(time.Now().UnixNano())
	receipt := builders.TransactionReceipt().WithRandomHash(ctrlRand).Build()

	response := toGetTxStatusOutput(cfg, &txOutput{
		transactionStatus:  protocol.TRANSACTION_STATUS_COMMITTED,
		transactionReceipt: receipt,
		blockHeight:        126,
		blockTimestamp:     blockTime,
	})

	require.EqualValues(t, protocol.REQUEST_STATUS_COMPLETED, response.ClientResponse.RequestResult().RequestStatus(), "Request status is wrong")
	require.EqualValues(t, 126, response.ClientResponse.RequestResult().BlockHeight(), "Block height response is wrong")
	require.EqualValues(t, blockTime, response.ClientResponse.RequestResult().BlockTimestamp(), "Block time response is wrong")

	require.EqualValues(t, protocol.TRANSACTION_STATUS_COMMITTED, response.ClientResponse.TransactionStatus(), "txStatus response is wrong")
	test.RequireCmpEqual(t, receipt, response.ClientResponse.TransactionReceipt(), "Transaction receipt is not equal")
}

func TestGetTransactionStatus_PrepareResponse_NilReceipt(t *testing.T) {
	cfg := config.ForPublicApiTests(22, 0, time.Minute)
	blockTime := primitives.TimestampNano(time.Now().UnixNano())

	response := toGetTxStatusOutput(cfg, &txOutput{
		transactionStatus:  protocol.TRANSACTION_STATUS_PENDING,
		transactionReceipt: nil,
		blockHeight:        8,
		blockTimestamp:     blockTime,
	})

	require.EqualValues(t, protocol.REQUEST_STATUS_IN_PROCESS, response.ClientResponse.RequestResult().RequestStatus(), "Request status is wrong")
	require.EqualValues(t, 8, response.ClientResponse.RequestResult().BlockHeight(), "Block height response is wrong")
	require.EqualValues(t, blockTime, response.ClientResponse.RequestResult().BlockTimestamp(), "Block time response is wrong")

	require.EqualValues(t, protocol.TRANSACTION_STATUS_PENDING, response.ClientResponse.TransactionStatus(), "txStatus response is wrong")
	require.Equal(t, 0, len(response.ClientResponse.TransactionReceipt().Raw()), "Transaction receipt is not equal")
	test.RequireCmpEqual(t, (*protocol.TransactionReceiptBuilder)(nil).Build(), response.ClientResponse.TransactionReceipt(), "Transaction receipt is not equal")
}

func TestGetTransactionStatus_PrepareResponse_OutOfSync(t *testing.T) {
	ctrlRand := rand.NewControlledRand(t)
	cfg := config.ForPublicApiTests(22, 0, time.Minute)
	blockTime := primitives.TimestampNano(time.Now().Add(time.Hour * -1).UnixNano())
	receipt := builders.TransactionReceipt().WithRandomHash(ctrlRand).Build()

	response := toGetTxStatusOutput(cfg, &txOutput{
		transactionStatus:  protocol.TRANSACTION_STATUS_COMMITTED,
		transactionReceipt: receipt,
		blockHeight:        126,
		blockTimestamp:     blockTime,
	})

	require.EqualValues(t, protocol.REQUEST_STATUS_COMPLETED, response.ClientResponse.RequestResult().RequestStatus(), "Request status is wrong")
	require.EqualValues(t, 126, response.ClientResponse.RequestResult().BlockHeight(), "Block height response is wrong")
	require.EqualValues(t, blockTime, response.ClientResponse.RequestResult().BlockTimestamp(), "Block time response is wrong")

	require.EqualValues(t, protocol.TRANSACTION_STATUS_COMMITTED, response.ClientResponse.TransactionStatus(), "txStatus response is wrong")
	test.RequireCmpEqual(t, receipt, response.ClientResponse.TransactionReceipt(), "Transaction receipt is not equal")
}

func TestGetTransactionStatus_PrepareResponse_NilReceipt_OutOfSync(t *testing.T) {
	cfg := config.ForPublicApiTests(22, 0, time.Minute)
	blockTime := primitives.TimestampNano(time.Now().Add(time.Hour * -1).UnixNano())

	response := toGetTxStatusOutput(cfg, &txOutput{
		transactionStatus:  protocol.TRANSACTION_STATUS_PENDING,
		transactionReceipt: nil,
		blockHeight:        8,
		blockTimestamp:     blockTime,
	})

	require.EqualValues(t, protocol.REQUEST_STATUS_OUT_OF_SYNC, response.ClientResponse.RequestResult().RequestStatus(), "Request status is wrong")
	require.EqualValues(t, 8, response.ClientResponse.RequestResult().BlockHeight(), "Block height response is wrong")
	require.EqualValues(t, blockTime, response.ClientResponse.RequestResult().BlockTimestamp(), "Block time response is wrong")

	require.EqualValues(t, protocol.TRANSACTION_STATUS_PENDING, response.ClientResponse.TransactionStatus(), "txStatus response is wrong")
	require.Equal(t, 0, len(response.ClientResponse.TransactionReceipt().Raw()), "Transaction receipt is not equal")
	test.RequireCmpEqual(t, (*protocol.TransactionReceiptBuilder)(nil).Build(), response.ClientResponse.TransactionReceipt(), "Transaction receipt is not equal")
}
