// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package publicapi

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGetBlock_PrepareResponse(t *testing.T) {
	blockTime := time.Now()
	blockPair := builders.BlockPair().WithHeight(88).WithBlockCreated(blockTime).Build()

	response := toGetBlockOutput(blockPair)

	require.EqualValues(t, protocol.REQUEST_STATUS_COMPLETED, response.ClientResponse.RequestResult().RequestStatus(), "Request status is wrong")
	require.EqualValues(t, 88, response.ClientResponse.RequestResult().BlockHeight(), "Block height response is wrong")
	require.EqualValues(t, blockTime.UnixNano(), response.ClientResponse.RequestResult().BlockTimestamp(), "Block time response is wrong")
	require.EqualValues(t, blockPair.TransactionsBlock.Header.Raw(), response.ClientResponse.TransactionsBlockHeader().Raw())
	require.EqualValues(t, blockPair.TransactionsBlock.Metadata.Raw(), response.ClientResponse.TransactionsBlockMetadata().Raw())
	require.EqualValues(t, blockPair.TransactionsBlock.BlockProof.Raw(), response.ClientResponse.TransactionsBlockProof().Raw())
	// TODO v1 how to check list of ...
	//require.Equal(t, len(blockPair.TransactionsBlock.SignedTransactions), len(response.ClientResponse.RawSignedTransactionsArray()))
	require.EqualValues(t, blockPair.ResultsBlock.Header.Raw(), response.ClientResponse.ResultsBlockHeader().Raw())
	require.EqualValues(t, blockPair.ResultsBlock.BlockProof.Raw(), response.ClientResponse.ResultsBlockProof().Raw())
	//require.EqualValues(t, len(blockPair.ResultsBlock.TransactionReceipts), len(response.ClientResponse.RawTransactionReceiptsArray()))
	//require.EqualValues(t, len(blockPair.ResultsBlock.ContractStateDiffs), len(response.ClientResponse.RawContractStateDiffsArray()))
}

func TestGetBlock_PrepareResponse_BadRequest(t *testing.T) {
	blockTime := primitives.TimestampNano(time.Now().Nanosecond())
	response := toGetBlockErrOutput(protocol.REQUEST_STATUS_BAD_REQUEST, 4, blockTime)

	require.EqualValues(t, protocol.REQUEST_STATUS_BAD_REQUEST, response.ClientResponse.RequestResult().RequestStatus(), "Request status is wrong")
	require.EqualValues(t, 4, response.ClientResponse.RequestResult().BlockHeight(), "Block height response is wrong")
	require.EqualValues(t, blockTime, response.ClientResponse.RequestResult().BlockTimestamp(), "Block time response is wrong")
	require.Len(t, response.ClientResponse.TransactionsBlockHeader().Raw(), 0)
	require.Len(t, response.ClientResponse.TransactionsBlockMetadata().Raw(), 0)
	require.Len(t, response.ClientResponse.TransactionsBlockProof().Raw(), 0)
	require.Len(t, response.ClientResponse.ResultsBlockHeader().Raw(), 0)
	require.Len(t, response.ClientResponse.ResultsBlockProof().Raw(), 0)
}

func TestGetBlock_PrepareResponse_SystemErr(t *testing.T) {
	response := toGetBlockErrOutput(protocol.REQUEST_STATUS_SYSTEM_ERROR, 0, 0)

	require.EqualValues(t, protocol.REQUEST_STATUS_SYSTEM_ERROR, response.ClientResponse.RequestResult().RequestStatus(), "Request status is wrong")
	require.EqualValues(t, 0, response.ClientResponse.RequestResult().BlockHeight(), "Block height response is wrong")
	require.EqualValues(t, 0, response.ClientResponse.RequestResult().BlockTimestamp(), "Block time response is wrong")
	require.Len(t, response.ClientResponse.TransactionsBlockHeader().Raw(), 0)
	require.Len(t, response.ClientResponse.TransactionsBlockMetadata().Raw(), 0)
	require.Len(t, response.ClientResponse.TransactionsBlockProof().Raw(), 0)
	require.Len(t, response.ClientResponse.ResultsBlockHeader().Raw(), 0)
	require.Len(t, response.ClientResponse.ResultsBlockProof().Raw(), 0)
}
