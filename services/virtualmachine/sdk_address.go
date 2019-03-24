// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package virtualmachine

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

func (s *service) handleSdkAddressCall(ctx context.Context, executionContext *executionContext, methodName primitives.MethodName, args []*protocol.Argument, permissionScope protocol.ExecutionPermissionScope) ([]*protocol.Argument, error) {
	switch methodName {

	case "getSignerAddress":
		value, err := s.handleSdkAddressGetSignerAddress(executionContext, args)
		if err != nil {
			return nil, err
		}
		return []*protocol.Argument{(&protocol.ArgumentBuilder{
			// value
			Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
			BytesValue: value,
		}).Build()}, nil

	case "getCallerAddress":
		value, err := s.handleSdkAddressGetCallerAddress(executionContext, args)
		if err != nil {
			return nil, err
		}
		return []*protocol.Argument{(&protocol.ArgumentBuilder{
			// value
			Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
			BytesValue: value,
		}).Build()}, nil

	case "getOwnAddress":
		value, err := s.handleSdkAddressGetOwnAddress(executionContext, args)
		if err != nil {
			return nil, err
		}
		return []*protocol.Argument{(&protocol.ArgumentBuilder{
			// value
			Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
			BytesValue: value,
		}).Build()}, nil

	default:
		return nil, errors.Errorf("unknown SDK address call method: %s", methodName)
	}
}

// outputArg0: value ([]byte)
func (s *service) handleSdkAddressGetSignerAddress(executionContext *executionContext, args []*protocol.Argument) ([]byte, error) {
	if len(args) != 0 {
		return nil, errors.Errorf("invalid SDK address getSignerAddress args: %v", args)
	}

	if executionContext.transactionOrQuery == nil {
		return nil, errors.New("operation does not contain a transaction or query")
	}

	return s.getSignerAddress(executionContext.transactionOrQuery.Signer())
}

// outputArg0: value ([]byte)
func (s *service) handleSdkAddressGetCallerAddress(executionContext *executionContext, args []*protocol.Argument) ([]byte, error) {
	if len(args) != 0 {
		return nil, errors.Errorf("invalid SDK address getCallerAddress args: %v", args)
	}

	if executionContext.serviceStackDepth() == 1 {
		// on the first caller, fallback to GetSignerAddress
		return s.handleSdkAddressGetSignerAddress(executionContext, args)
	} else {
		// after a contract call, get the caller address
		return digest.CalcClientAddressOfContract(executionContext.serviceStackPeekCaller())
	}
}

// outputArg0: value ([]byte)
func (s *service) handleSdkAddressGetOwnAddress(executionContext *executionContext, args []*protocol.Argument) ([]byte, error) {
	if len(args) != 0 {
		return nil, errors.Errorf("invalid SDK address getOwnAddress args: %v", args)
	}

	return digest.CalcClientAddressOfContract(executionContext.serviceStackPeekCurrent())
}
