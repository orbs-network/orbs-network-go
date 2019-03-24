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

func exampleEventSignature(name string, amount uint64) {}

func TestSdkEvents_EmitEvent(t *testing.T) {
	s := createEventsSdk()

	require.NotPanics(t, func() {
		s.SdkEventsEmitEvent(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SERVICE, exampleEventSignature, "hello", uint64(17))
	}, "should not panic on happy flow")
}

func TestSdkEvents_EmitEvent_NotAFunctionSignature(t *testing.T) {
	s := createEventsSdk()

	require.Panics(t, func() {
		s.SdkEventsEmitEvent(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SERVICE, "OopsNotAFunction", uint64(17), "hello")
	}, "should panic because not a valid function")
}

func TestSdkEvents_EmitEvent_FunctionSignatureWrongNumOfArgs(t *testing.T) {
	s := createEventsSdk()

	require.Panics(t, func() {
		s.SdkEventsEmitEvent(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SERVICE, exampleEventSignature, "hello")
	}, "should panic because not enough args given")
}

func TestSdkEvents_EmitEvent_FunctionSignatureWrongTypes(t *testing.T) {
	s := createEventsSdk()

	require.Panics(t, func() {
		s.SdkEventsEmitEvent(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SERVICE, exampleEventSignature, uint64(17), "hello")
	}, "should panic because wrong types given")
}

func createEventsSdk() *service {
	return &service{sdkHandler: &contractSdkEventsCallHandlerStub{}}
}

type contractSdkEventsCallHandlerStub struct{}

func (c *contractSdkEventsCallHandlerStub) HandleSdkCall(ctx context.Context, input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
	if input.PermissionScope != protocol.PERMISSION_SCOPE_SERVICE {
		panic("permissions passed to SDK are incorrect")
	}
	switch input.MethodName {
	case "emitEvent":
		return nil, nil
	default:
		return nil, errors.New("unknown method")
	}
}
