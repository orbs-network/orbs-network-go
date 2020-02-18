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

const CALL_COMMITTEE_CONTRACT_INTERVAL = 200 * time.Millisecond

func (s *service) getOrderedCommittee(ctx context.Context, currentBlockHeight primitives.BlockHeight) ([]primitives.NodeAddress, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	// current block is used as seed and needs to be for the block being calculated Now.
	logger.Info("system-call GetOrderedCommittee", logfields.BlockHeight(currentBlockHeight), log.Stringable("interval-between-attempts", CALL_COMMITTEE_CONTRACT_INTERVAL))
	orderedCommittee, err := s.callGetOrderedCommitteeSystemContractUntilSuccess(ctx, currentBlockHeight)
	if err != nil {
		return nil, err
	}
	logger.Info("system-call elected validators", log.Int("num-results", len(orderedCommittee)), logfields.BlockHeight(currentBlockHeight))

	return orderedCommittee, nil
}

func (s *service) callGetOrderedCommitteeSystemContractUntilSuccess(ctx context.Context, blockHeight primitives.BlockHeight) ([]primitives.NodeAddress, error) {
	attempts := 1
	for {
		// exit on system shutdown
		if ctx.Err() != nil {
			return nil, errors.New("context terminated during callGetOrderedCommitteeSystemContractUntilSuccess")
		}

		orderedCommittee, err := s.callGetOrderedCommitteeSystemContract(ctx, blockHeight)
		if err == nil {
			return orderedCommittee, nil
		}

		// log every 500 failures
		if attempts%500 == 1 {
			if ctx.Err() == nil { // this may fail rightfully on graceful shutdown (ctx.Done), we don't want to report an error in this case
				s.logger.Info("cannot get ordered committee from system contract", log.Error(err), logfields.BlockHeight(blockHeight), log.Int("attempts", attempts))
			}
		}

		// sleep or wait for ctx done, whichever comes first
		sleepOrShutdown, cancel := context.WithTimeout(ctx, CALL_COMMITTEE_CONTRACT_INTERVAL)
		<-sleepOrShutdown.Done()
		cancel()

		attempts++
	}
}

func (s *service) callGetOrderedCommitteeSystemContract(ctx context.Context, blockHeight primitives.BlockHeight) ([]primitives.NodeAddress, error) {
	systemContractName := primitives.ContractName(committee_systemcontract.CONTRACT_NAME)
	systemMethodName := primitives.MethodName(committee_systemcontract.METHOD_GET_ORDERED_COMMITTEE)

	output, err := s.virtualMachine.CallSystemContract(ctx, &services.CallSystemContractInput{
		BlockHeight:        blockHeight,
		BlockTimestamp:     primitives.TimestampNano(time.Now().UnixNano()), // use now as the call is a kind of RunQuery and doesn't happen under consensus
		ContractName:       systemContractName,
		MethodName:         systemMethodName,
		InputArgumentArray: protocol.ArgumentsArrayEmpty(),
	})
	if err != nil {
		return nil, err
	}
	if output.CallResult != protocol.EXECUTION_RESULT_SUCCESS {
		return nil, errors.Errorf("call system %s.%s call result is %s", systemContractName, systemMethodName, output.CallResult)
	}

	argIterator := output.OutputArgumentArray.ArgumentsIterator()
	if !argIterator.HasNext() {
		return nil, errors.Errorf("call system %s.%s returned corrupt output value", systemContractName, systemMethodName)
	}
	arg0 := argIterator.NextArguments()
	if !arg0.IsTypeBytesArrayValue() {
		return nil, errors.Errorf("call system %s.%s returned corrupt output value", systemContractName, systemMethodName)
	}
	return toAddresses(arg0), nil
}

// helper
func toAddresses(input *protocol.Argument) (addresses []primitives.NodeAddress) {
	itr := input.BytesArrayValueIterator()
	for itr.HasNext() {
		addresses = append(addresses, itr.NextBytes())
	}
	return
}
