// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package publicapi

import (
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRunQuery_PrepareResponse(t *testing.T) {
	blockTime := primitives.TimestampNano(time.Now().UnixNano())
	outputArgs := builders.PackedArgumentArrayEncode("hello", uint64(17))
	outputEvents := builders.PackedEventsArrayEncode(nil)

	response := toRunQueryOutput(&queryOutput{
		callOutput: &services.ProcessQueryOutput{
			CallResult:              protocol.EXECUTION_RESULT_SUCCESS,
			OutputArgumentArray:     outputArgs,
			OutputEventsArray:       outputEvents,
			ReferenceBlockHeight:    126,
			ReferenceBlockTimestamp: blockTime,
		},
	})

	require.EqualValues(t, protocol.REQUEST_STATUS_COMPLETED, response.ClientResponse.RequestResult().RequestStatus(), "Request status is wrong")
	require.EqualValues(t, 126, response.ClientResponse.RequestResult().BlockHeight(), "Block height response is wrong")
	require.EqualValues(t, blockTime, response.ClientResponse.RequestResult().BlockTimestamp(), "Block time response is wrong")

	require.EqualValues(t, protocol.EXECUTION_RESULT_SUCCESS, response.ClientResponse.QueryResult().ExecutionResult(), "Execution result is wrong")
	test.RequireCmpEqual(t, outputArgs, response.ClientResponse.QueryResult().OutputArgumentArray(), "OutputArgs not equal")
	test.RequireCmpEqual(t, outputEvents, response.ClientResponse.QueryResult().OutputEventsArray(), "OutputEvents not equal")
}

func TestRunQuery_PrepareResponse_NilExecution(t *testing.T) {
	response := toRunQueryOutput(&queryOutput{
		requestStatus: protocol.REQUEST_STATUS_BAD_REQUEST,
	})

	require.EqualValues(t, protocol.REQUEST_STATUS_BAD_REQUEST, response.ClientResponse.RequestResult().RequestStatus(), "Request status is wrong")
	require.EqualValues(t, 0, response.ClientResponse.RequestResult().BlockHeight(), "Block height response is wrong")
	require.EqualValues(t, 0, response.ClientResponse.RequestResult().BlockTimestamp(), "Block time response is wrong")

	require.Equal(t, 0, len(response.ClientResponse.QueryResult().Raw()), "Query result is not equal")
	test.RequireCmpEqual(t, (*protocol.QueryResultBuilder)(nil).Build(), response.ClientResponse.QueryResult(), "Query result is not equal")
}
