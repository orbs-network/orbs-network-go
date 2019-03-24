// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package native

import (
	"context"
	"encoding/hex"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

var exampleAddress1, _ = hex.DecodeString("1acb19a469206161ed7e5ed9feb996a6e24be441")
var exampleAddress2, _ = hex.DecodeString("223344a469206161ed7e5ed9feb996a6e24be441")
var exampleAddress3, _ = hex.DecodeString("33ee44a469206161ed7e5ed9feb996a6e24be441")

func TestSdkAddress_GetSignerAddress(t *testing.T) {
	s := createAddressSdk()

	address := s.SdkAddressGetSignerAddress(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SERVICE)
	require.EqualValues(t, exampleAddress1, address, "example1 should be returned")
}

func TestSdkAddress_GetCallerAddress(t *testing.T) {
	s := createAddressSdk()

	address := s.SdkAddressGetCallerAddress(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SERVICE)
	require.EqualValues(t, exampleAddress2, address, "example2 should be returned")
}

func TestSdkAddress_GetOwnAddress(t *testing.T) {
	s := createAddressSdk()

	address := s.SdkAddressGetOwnAddress(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SERVICE)
	require.EqualValues(t, exampleAddress3, address, "example3 should be returned")
}

func TestSdkAddress_GetContractAddress(t *testing.T) {
	s := createAddressSdk()

	address := s.SdkAddressGetContractAddress(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SERVICE, "SomeContract")
	require.EqualValues(t, digest.CLIENT_ADDRESS_SIZE_BYTES, len(address), "a valid address should be returned")
}

func createAddressSdk() *service {
	return &service{sdkHandler: &contractSdkAddressCallHandlerStub{}}
}

type contractSdkAddressCallHandlerStub struct{}

func (c *contractSdkAddressCallHandlerStub) HandleSdkCall(ctx context.Context, input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
	if input.PermissionScope != protocol.PERMISSION_SCOPE_SERVICE {
		panic("permissions passed to SDK are incorrect")
	}
	switch input.MethodName {
	case "getSignerAddress":
		return &handlers.HandleSdkCallOutput{
			OutputArguments: builders.Arguments(exampleAddress1),
		}, nil
	case "getCallerAddress":
		return &handlers.HandleSdkCallOutput{
			OutputArguments: builders.Arguments(exampleAddress2),
		}, nil
	case "getOwnAddress":
		return &handlers.HandleSdkCallOutput{
			OutputArguments: builders.Arguments(exampleAddress3),
		}, nil
	default:
		return nil, errors.New("unknown method")
	}
}
