// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package publicapi

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"time"
)

func (s *service) RunQuery(parentCtx context.Context, input *services.RunQueryInput) (*services.RunQueryOutput, error) {
	ctx := trace.NewContext(parentCtx, "PublicApi.RunQuery")

	if input.ClientRequest == nil {
		err := errors.Errorf("client request is nil")
		s.logger.Info("run query received missing input", log.Error(err))
		return nil, err
	}

	query := input.ClientRequest.SignedQuery().Query()
	queryHash := digest.CalcQueryHash(query)
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx), log.Query(queryHash), log.String("flow", "checkpoint"))

	if _, err := validateRequest(s.config, query.ProtocolVersion(), query.VirtualChainId()); err != nil {
		logger.Info("run query received input failed", log.Error(err))
		return toRunQueryOutput(&queryOutput{requestStatus: protocol.REQUEST_STATUS_BAD_REQUEST}), err
	}

	logger.Info("run query request received")

	start := time.Now()
	defer s.metrics.runQueryTime.RecordSince(start)

	callOutput, err := s.virtualMachine.ProcessQuery(ctx, &services.ProcessQueryInput{
		BlockHeight: 0, // recent block height
		SignedQuery: input.ClientRequest.SignedQuery(),
	})
	if err != nil {
		logger.Info("run query request failed", log.Error(err))
		return toRunQueryOutput(&queryOutput{callOutput: callOutput}), err
	}

	return toRunQueryOutput(&queryOutput{callOutput: callOutput}), nil
}

func toRunQueryOutput(out *queryOutput) *services.RunQueryOutput {
	response := &client.RunQueryResponseBuilder{
		RequestResult: &client.RequestResultBuilder{
			RequestStatus: out.requestStatus,
		},
		QueryResult: nil,
	}
	if out.callOutput != nil {
		response.RequestResult.RequestStatus = translateExecutionStatusToRequestStatus(out.callOutput.CallResult)
		response.RequestResult.BlockHeight = out.callOutput.ReferenceBlockHeight
		response.RequestResult.BlockTimestamp = out.callOutput.ReferenceBlockTimestamp
		response.QueryResult = &protocol.QueryResultBuilder{
			ExecutionResult:     out.callOutput.CallResult,
			OutputArgumentArray: out.callOutput.OutputArgumentArray,
			OutputEventsArray:   out.callOutput.OutputEventsArray,
		}
	}
	return &services.RunQueryOutput{ClientResponse: response.Build()}
}
