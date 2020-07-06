// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package virtualmachine

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
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

	case "getNextBlockCommittee":
		value, err := s.handleSdkEnvGetNextBlockCommittee(ctx, executionContext, args)
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
	if len(executionContext.transactionOrQuery.Signer().Raw()) != 0 {
		return []byte{}, errors.New("invalid call to SDK env getBlockProposerAddress can only be called by system contract")
	}

	return executionContext.currentBlockProposerAddress, nil
}

// outputArg0: value array of (bytes)
func (s *service) handleSdkEnvGetBlockCommittee(ctx context.Context, executionContext *executionContext, args []*protocol.Argument) ([][]byte, error) {
	if len(args) != 0 {
		return [][]byte{}, errors.Errorf("invalid SDK env getBlockCommittee args: %v", args)
	}
	//if len(executionContext.transactionOrQuery.Signer().Raw()) != 0 {
	//	return [][]byte{}, errors.New("invalid call to SDK env getBlockCommittee can only be called by system contract")
	//}

	res, err := s.management.GetCommittee(ctx, &services.GetCommitteeInput{Reference: executionContext.lastBlockReferenceTime})
	if err != nil {
		s.logger.Error("management.GetCommittee failed", log.Error(err))
		return [][]byte{}, err
	}
	var committee [][]byte
	for _, c := range res.Members {
		committee = append(committee, c)
	}
	return committee, nil
}

// outputArg0: value array of (bytes)
func (s *service) handleSdkEnvGetNextBlockCommittee(ctx context.Context, executionContext *executionContext, args []*protocol.Argument) ([][]byte, error) {
	if len(args) != 0 {
		return [][]byte{}, errors.Errorf("invalid SDK env getNextBlockCommittee args: %v", args)
	}
	if len(executionContext.transactionOrQuery.Signer().Raw()) != 0 {
		return [][]byte{}, errors.New("invalid call to SDK env getNextBlockCommittee can only be called by system contract")
	}

	res, err := s.management.GetCommittee(ctx, &services.GetCommitteeInput{Reference: executionContext.currentBlockReferenceTime})
	if err != nil {
		s.logger.Error("management.GetCommittee failed", log.Error(err))
		return [][]byte{}, err
	}
	var committee [][]byte
	for _, c := range res.Members {
		committee = append(committee, c)
	}
	return committee, err
}
