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

// this example represents respones from "say" function with data "etherworld"
var examplePackedOutput = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 16, 104, 101, 108, 108, 111, 32, 101, 116, 104, 101, 114, 119, 111, 114, 108, 100, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
var exampleBlockNumber = uint64(1234)
var exampleTxIndex = uint32(56)
var exampleBlockTimestamp = uint64(12345)

func TestSdkEthereum_CallMethod(t *testing.T) {
	s := createEthereumSdk()

	var out string
	sampleABI := `[{"inputs":[],"name":"say","outputs":[{"name":"","type":"string"}],"type":"function"}]`
	s.SdkEthereumCallMethod(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SYSTEM, "ExampleAddress", sampleABI, exampleBlockNumber, "say", &out)

	require.Equal(t, "hello etherworld", out, "did not get the expected return value from ethereum call")
}

func TestSdkEthereum_GetTransactionLog(t *testing.T) {
	s := createEthereumSdk()

	var out string
	sampleABI := `[{"inputs":[{"name":"sentence","type":"string"}],"name":"said","type":"event"}]`
	ethBlockNumber, ethTxIndex := s.SdkEthereumGetTransactionLog(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SYSTEM, "ExampleAddress", sampleABI, "0x88df016429689c079f3b2f6ad39fa052532c56795b733da78a91ebe6a713944b", "said", &out)

	require.Equal(t, "hello etherworld", out, "did not get the expected return value from transaction log")
	require.Equal(t, exampleBlockNumber, ethBlockNumber, "did not get expected block number from transaction log")
	require.Equal(t, exampleTxIndex, ethTxIndex, "did not get expected txIndex from transaction log")
}

func TestSdkEthereum_GetBlockNumber(t *testing.T) {
	s := createEthereumSdk()

	ethBlockNumber := s.SdkEthereumGetBlockNumber(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SYSTEM)

	require.Equal(t, exampleBlockNumber, ethBlockNumber, "did not get expected block number from Ethereum")
}

func TestSdkEthereum_GetBlockNumberByTime(t *testing.T) {
	s := createEthereumSdk()

	ethBlockNumber := s.SdkEthereumGetBlockNumberByTime(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SYSTEM, exampleBlockTimestamp)

	require.Equal(t, exampleBlockNumber, ethBlockNumber, "did not get expected block number from Ethereum")
}

func TestSdkEthereum_GetBlockTime(t *testing.T) {
	s := createEthereumSdk()

	ethBlockTimestamp := s.SdkEthereumGetBlockTime(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SYSTEM)

	require.Equal(t, exampleBlockTimestamp, ethBlockTimestamp, "did not get expected block time from Ethereum")
}

func TestSdkEthereum_GetBlockTimeByNumber(t *testing.T) {
	s := createEthereumSdk()

	ethBlockTimestamp := s.SdkEthereumGetBlockTimeByNumber(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SYSTEM, exampleBlockNumber)

	require.Equal(t, exampleBlockTimestamp, ethBlockTimestamp, "did not get expected block number from Ethereum")
}

func createEthereumSdk() *service {
	return &service{sdkHandler: &contractSdkEthereumCallHandlerStub{}}
}

type contractSdkEthereumCallHandlerStub struct{}

func (c *contractSdkEthereumCallHandlerStub) HandleSdkCall(ctx context.Context, input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
	if input.PermissionScope != protocol.PERMISSION_SCOPE_SYSTEM {
		panic("permissions passed to SDK are incorrect")
	}
	outputNatives := make([]interface{}, 0)
	switch input.MethodName {
	case "callMethod":
		outputNatives = append(outputNatives, examplePackedOutput)
	case "getTransactionLog":
		outputNatives = append(outputNatives, examplePackedOutput, exampleBlockNumber, exampleTxIndex)
	case "getBlockNumber":
		outputNatives = append(outputNatives, exampleBlockNumber)
	case "getBlockNumberByTime":
		outputNatives = append(outputNatives, exampleBlockNumber)
	case "getBlockTime":
		outputNatives = append(outputNatives, exampleBlockTimestamp)
	case "getBlockTimeByNumber":
		outputNatives = append(outputNatives, exampleBlockTimestamp)
	default:
		return nil, errors.New("unknown method")
	}
	outputArgs, err := protocol.ArgumentsFromNatives(outputNatives)
	if err != nil {
		return nil, errors.Wrapf(err, "unknown input arg")
	}
	return &handlers.HandleSdkCallOutput{OutputArguments: outputArgs}, nil
}
