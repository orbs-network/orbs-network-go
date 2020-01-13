// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package virtualmachine

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	elections_systemcontract "github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Elections"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) handleSdkEnvCall(ctx context.Context, executionContext *executionContext, methodName primitives.MethodName, args []*protocol.Argument, permissionScope protocol.ExecutionPermissionScope) ([]*protocol.Argument, error) {
	switch methodName {

	case "getBlockHeight":
		value, err := s.handleSdkEnvGetBlockHeight(executionContext, args)
		if err != nil {
			return nil, err
		}
		return []*protocol.Argument{(&protocol.ArgumentBuilder{
			// value
			Type:        protocol.ARGUMENT_TYPE_UINT_64_VALUE,
			Uint64Value: value,
		}).Build()}, nil

	case "getBlockTimestamp":
		value, err := s.handleSdkEnvGetBlockTimestamp(executionContext, args)
		if err != nil {
			return nil, err
		}
		return []*protocol.Argument{(&protocol.ArgumentBuilder{
			// value
			Type:        protocol.ARGUMENT_TYPE_UINT_64_VALUE,
			Uint64Value: value,
		}).Build()}, nil

	case "getBlockProposerAddress":
		value, err := s.handleSdkEnvGetBlockProposerAddress(executionContext, args)
		if err != nil {
			return nil, err
		}
		return []*protocol.Argument{(&protocol.ArgumentBuilder{
			// value
			Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
			BytesValue: value,
		}).Build()}, nil

	case "getBlockCommittee":
		value, err := s.handleSdkEnvGetBlockCommittee(ctx, executionContext, args)
		if err != nil {
			return nil, err
		}
		return []*protocol.Argument{(&protocol.ArgumentBuilder{
			// value
			Type:            protocol.ARGUMENT_TYPE_BYTES_ARRAY_VALUE,
			BytesArrayValue: value,
		}).Build()}, nil

	default:
		return nil, errors.Errorf("unknown SDK env call method: %s", methodName)
	}
}

// outputArg0: value (uint64)
func (s *service) handleSdkEnvGetBlockHeight(executionContext *executionContext, args []*protocol.Argument) (uint64, error) {
	if len(args) != 0 {
		return 0, errors.Errorf("invalid SDK env getBlockHeight args: %v", args)
	}

	return uint64(executionContext.currentBlockHeight), nil
}

// outputArg0: value (uint64)
func (s *service) handleSdkEnvGetBlockTimestamp(executionContext *executionContext, args []*protocol.Argument) (uint64, error) {
	if len(args) != 0 {
		return 0, errors.Errorf("invalid SDK env getBlockTimestamp args: %v", args)
	}

	return uint64(executionContext.currentBlockTimestamp), nil
}

// outputArg0: value (bytes)
func (s *service) handleSdkEnvGetBlockProposerAddress(executionContext *executionContext, args []*protocol.Argument) ([]byte, error) {
	if len(args) != 0 {
		return []byte{}, errors.Errorf("invalid SDK env getBlockProposerAddress args: %v", args)
	}

	return executionContext.currentBlockProposerAddress, nil
}

// outputArg0: value array of (bytes)
func (s *service) handleSdkEnvGetBlockCommittee(ctx context.Context, executionContext *executionContext, args []*protocol.Argument) ([][]byte, error) {
	// TODO POSV2 should get ref as input ?
	if len(args) != 0 {
		return [][]byte{}, errors.Errorf("invalid SDK env getBlockProposerAddress args: %v", args)
	}

	// TODO POSV2 should be seperate provider ?
	var committeeNodeAddresses []primitives.NodeAddress
	var err error
	//committeeNodeAddresses, err := s.callElectionsSystemContract(ctx, executionContext.currentBlockHeight)

	if err != nil || len(committeeNodeAddresses) == 0 {
		committeeNodeAddresses, err = s.committeeProvider.GetCommittee(ctx, uint64(executionContext.currentBlockHeight))
	}
	var committee [][]byte
	for _, c := range committeeNodeAddresses {
		committee = append(committee, c)
	}
	return committee, err
}

func (s *service) callElectionsSystemContract(ctx context.Context, blockHeight primitives.BlockHeight) ([]primitives.NodeAddress, error) {
	systemContractName := primitives.ContractName(elections_systemcontract.CONTRACT_NAME)
	systemMethodName := primitives.MethodName(elections_systemcontract.METHOD_GET_ELECTED_VALIDATORS)

	output, err := s.CallSystemContract(ctx, &services.CallSystemContractInput{
		BlockHeight:        blockHeight,
		BlockTimestamp:     0, // unfortunately we don't know the timestamp here, this limits which contract SDK API can be used
		ContractName:       systemContractName,
		MethodName:         systemMethodName,
		InputArgumentArray: protocol.ArgumentsArrayEmpty(),
	})
	if err != nil {
		return nil, err
	}
	if output.CallResult != protocol.EXECUTION_RESULT_SUCCESS {
		return nil, errors.Errorf("%s.%s call result is %s", systemContractName, systemMethodName, output.CallResult)
	}

	argIterator := output.OutputArgumentArray.ArgumentsIterator()
	if !argIterator.HasNext() {
		return nil, errors.Errorf("call system %s.%s returned corrupt output value", systemContractName, systemMethodName)
	}
	arg0 := argIterator.NextArguments()
	if !arg0.IsTypeBytesValue() {
		return nil, errors.Errorf("call system %s.%s returned corrupt output value", systemContractName, systemMethodName)
	}
	joinedAddresses := arg0.BytesValue()

	numAddresses := len(joinedAddresses) / digest.NODE_ADDRESS_SIZE_BYTES
	res := make([]primitives.NodeAddress, numAddresses)
	for i := 0; i < numAddresses; i++ {
		res[i] = joinedAddresses[20*i : 20*(i+1)]
	}
	return res, nil
}
