// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package native

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSdkService_CallMethod(t *testing.T) {
	s := createServiceSdk()

	res := s.SdkServiceCallMethod(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SYSTEM, "AnotherContract", "someMethod", uint64(17), "hello")
	require.Equal(t, []interface{}{uint64(17), "hello"}, res, "callMethod result should match expected")
}

func TestSdkService_CallMethod_FailingCall(t *testing.T) {
	s := createServiceSdk()

	require.Panics(t, func() {
		s.SdkServiceCallMethod(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SYSTEM, "AnotherFailingContract", "someMethod", uint64(17), "hello")
	}, "should panic because the call failed (called contract threw an exception)")
}

func createServiceSdk() *service {
	return &service{sdkHandler: &contractSdkServiceCallHandlerStub{}}
}

type contractSdkServiceCallHandlerStub struct{}

func (c *contractSdkServiceCallHandlerStub) HandleSdkCall(ctx context.Context, input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
	if input.PermissionScope != protocol.PERMISSION_SCOPE_SYSTEM {
		panic("permissions passed to SDK are incorrect")
	}
	switch input.MethodName {
	case "callMethod":
		if input.InputArguments[0].StringValue() == "AnotherFailingContract" {
			return nil, errors.New("failing call")
		}
		// all other contracts should succeed
		return &handlers.HandleSdkCallOutput{
			OutputArguments: []*protocol.Argument{input.InputArguments[2]},
		}, nil
	default:
		return nil, errors.New("unknown method")
	}
}
