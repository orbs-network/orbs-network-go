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

var EXAMPLE_CONTEXT = []byte{0x17, 0x18}

func exampleKey() []byte {
	return []byte("example-key")
}

func TestSdkState_WriteReadBytesByAddress(t *testing.T) {
	s := createStateSdk()
	s.SdkStateWriteBytes(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SERVICE, exampleKey(), []byte{0x01, 0x02, 0x03})

	bytes := s.SdkStateReadBytes(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SERVICE, exampleKey())
	require.Equal(t, []byte{0x01, 0x02, 0x03}, bytes, "read should return what was written")
}

func createStateSdk() *service {
	return &service{sdkHandler: &contractSdkStateCallHandlerStub{
		store: make(map[string]*protocol.Argument),
	}}
}

type contractSdkStateCallHandlerStub struct {
	store map[string]*protocol.Argument
}

func (c *contractSdkStateCallHandlerStub) HandleSdkCall(ctx context.Context, input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
	if input.PermissionScope != protocol.PERMISSION_SCOPE_SERVICE {
		panic("permissions passed to SDK are incorrect")
	}
	switch input.MethodName {
	case "read":
		return &handlers.HandleSdkCallOutput{
			OutputArguments: []*protocol.Argument{c.store[string(input.InputArguments[0].BytesValue())]},
		}, nil
	case "write":
		c.store[string(input.InputArguments[0].BytesValue())] = input.InputArguments[1]
		return nil, nil
	default:
		return nil, errors.New("unknown method")
	}
}
