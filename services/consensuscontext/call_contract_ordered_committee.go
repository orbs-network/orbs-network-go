// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/Committee"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"time"
)

func (s *service) getOrderedCommittee(ctx context.Context, currentBlockHeight primitives.BlockHeight, prevBlockReferenceTime primitives.TimestampSeconds) ([]primitives.NodeAddress, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	// current block is used as seed and needs to be for the block being calculated Now.
	orderedCommittee, err := s.callGetOrderedCommitteeSystemContract(ctx, currentBlockHeight, prevBlockReferenceTime)
	if err != nil {
		return nil, err
	}
	logger.Info("system-call elected validators", log.Int("num-results", len(orderedCommittee)), logfields.BlockHeight(currentBlockHeight))

	return orderedCommittee, nil
}

func (s *service) callGetOrderedCommitteeSystemContract(ctx context.Context, blockHeight primitives.BlockHeight, prevBlockReferenceTime primitives.TimestampSeconds) ([]primitives.NodeAddress, error) {
	systemContractName := primitives.ContractName(committee_systemcontract.CONTRACT_NAME)
	systemMethodName := primitives.MethodName(committee_systemcontract.METHOD_GET_ORDERED_COMMITTEE)

	output, err := s.virtualMachine.CallSystemContract(ctx, &services.CallSystemContractInput{
		BlockHeight:               blockHeight,
		BlockTimestamp:            primitives.TimestampNano(time.Now().UnixNano()), // use now as the call is a kind of RunQuery and doesn't happen under consensus
		ContractName:              systemContractName,
		MethodName:                systemMethodName,
		CurrentBlockReferenceTime: 0, // future use ?
		PrevBlockReferenceTime:    prevBlockReferenceTime,
		InputArgumentArray:        protocol.ArgumentsArrayEmpty(),
	})
	if err != nil {
		return nil, err
	}
	if output.CallResult != protocol.EXECUTION_RESULT_SUCCESS {
		return nil, errors.Errorf("GetOrderedCommittee call system %s.%s call result is %s", systemContractName, systemMethodName, output.CallResult)
	}

	argIterator := output.OutputArgumentArray.ArgumentsIterator()
	if !argIterator.HasNext() {
		return nil, errors.Errorf("GetOrderedCommittee call system %s.%s returned corrupt output value", systemContractName, systemMethodName)
	}
	arg0 := argIterator.NextArguments()
	if !arg0.IsTypeBytesArrayValue() {
		return nil, errors.Errorf("GetOrderedCommittee call system %s.%s returned corrupt output value", systemContractName, systemMethodName)
	}
	return toAddresses(arg0), nil
}

