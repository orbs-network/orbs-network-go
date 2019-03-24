// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"bytes"
	"context"
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func (h *harness) verifyHandlerRegistrations(t *testing.T) {
	for key, processor := range h.processors {
		ok, err := processor.Verify()
		if !ok {
			t.Fatal("Did not register with processor", key.String(), ":", err)
		}
	}
}

func (h *harness) expectNativeContractMethodCalled(expectedContractName primitives.ContractName, expectedMethodName primitives.MethodName, contractFunction func(primitives.ExecutionContextId, *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error)) {
	contractMethodMatcher := func(i interface{}) bool {
		input, ok := i.(*services.ProcessCallInput)
		return ok &&
			input.ContractName == expectedContractName &&
			input.MethodName == expectedMethodName &&
			input.CallingPermissionScope == protocol.PERMISSION_SCOPE_SERVICE
	}

	h.processors[protocol.PROCESSOR_TYPE_NATIVE].When("ProcessCall", mock.Any, mock.AnyIf(fmt.Sprintf("Contract equals %s and Method %s and permissions are service", expectedContractName, expectedMethodName), contractMethodMatcher)).Call(func(ctx context.Context, input *services.ProcessCallInput) (*services.ProcessCallOutput, error) {
		callResult, outputArgsArray, err := contractFunction(input.ContextId, input.InputArgumentArray)
		return &services.ProcessCallOutput{
			OutputArgumentArray: outputArgsArray,
			CallResult:          callResult,
		}, err
	}).Times(1)
}

func (h *harness) expectNativeContractMethodCalledWithSystemPermissions(expectedContractName primitives.ContractName, expectedMethodName primitives.MethodName, contractFunction func(primitives.ExecutionContextId) (protocol.ExecutionResult, *protocol.ArgumentArray, error)) {
	contractMethodMatcher := func(i interface{}) bool {
		input, ok := i.(*services.ProcessCallInput)
		return ok &&
			input.ContractName == expectedContractName &&
			input.MethodName == expectedMethodName &&
			input.CallingPermissionScope == protocol.PERMISSION_SCOPE_SYSTEM
	}

	h.processors[protocol.PROCESSOR_TYPE_NATIVE].When("ProcessCall", mock.Any, mock.AnyIf(fmt.Sprintf("Contract equals %s and Method %s and permissions are system", expectedContractName, expectedMethodName), contractMethodMatcher)).Call(func(ctx context.Context, input *services.ProcessCallInput) (*services.ProcessCallOutput, error) {
		callResult, outputArgsArray, err := contractFunction(input.ContextId)
		return &services.ProcessCallOutput{
			OutputArgumentArray: outputArgsArray,
			CallResult:          callResult,
		}, err
	}).Times(1)
}

func (h *harness) expectNativeContractMethodNotCalled(expectedContractName primitives.ContractName, expectedMethodName primitives.MethodName) {
	contractMethodMatcher := func(i interface{}) bool {
		input, ok := i.(*services.ProcessCallInput)
		return ok &&
			input.ContractName == expectedContractName &&
			input.MethodName == expectedMethodName
	}

	h.processors[protocol.PROCESSOR_TYPE_NATIVE].When("ProcessCall", mock.Any, mock.AnyIf(fmt.Sprintf("Contract equals %s and Method %s", expectedContractName, expectedMethodName), contractMethodMatcher)).Return(nil, nil).Times(0)
}

func (h *harness) verifyNativeContractMethodCalled(t *testing.T) {
	ok, err := h.processors[protocol.PROCESSOR_TYPE_NATIVE].Verify()
	require.True(t, ok, "did not call processor: %v", err)
}

func (h *harness) expectSystemContractCalled(expectedContractName string, expectedMethodName string, returnError error, returnArgs ...interface{}) {
	contractMethodMatcher := func(i interface{}) bool {
		input, ok := i.(*services.ProcessCallInput)
		return ok &&
			string(input.ContractName) == expectedContractName &&
			string(input.MethodName) == expectedMethodName
	}

	callResult := protocol.EXECUTION_RESULT_SUCCESS
	if returnError != nil {
		callResult = protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT
	}
	outputToReturn := &services.ProcessCallOutput{
		OutputArgumentArray: builders.ArgumentsArray(returnArgs...),
		CallResult:          callResult,
	}

	h.processors[protocol.PROCESSOR_TYPE_NATIVE].When("ProcessCall", mock.Any, mock.AnyIf(fmt.Sprintf("Contract equals %s and Method %s", expectedContractName, expectedMethodName), contractMethodMatcher)).Return(outputToReturn, returnError).AtLeast(1)
}

func (h *harness) verifySystemContractCalled(t *testing.T) {
	ok, err := h.processors[protocol.PROCESSOR_TYPE_NATIVE].Verify()
	require.True(t, ok, "did not call processor for system contract: %v", err)
}

func (h *harness) expectNativeContractInfoRequested(expectedContractName primitives.ContractName, returnError error) {
	contractMatcher := func(i interface{}) bool {
		input, ok := i.(*services.GetContractInfoInput)
		return ok &&
			input.ContractName == expectedContractName
	}

	outputToReturn := &services.GetContractInfoOutput{
		PermissionScope: protocol.PERMISSION_SCOPE_SERVICE,
	}

	h.processors[protocol.PROCESSOR_TYPE_NATIVE].When("GetContractInfo", mock.Any, mock.AnyIf(fmt.Sprintf("Contract equals %s", expectedContractName), contractMatcher)).Return(outputToReturn, returnError).Times(1)
}

func (h *harness) verifyNativeContractInfoRequested(t *testing.T) {
	ok, err := h.processors[protocol.PROCESSOR_TYPE_NATIVE].Verify()
	require.True(t, ok, "did not request info for native contract: %v", err)
}

func (h *harness) expectEthereumConnectorMethodCalled(expectedContractAddress string, expectedBlockNumber uint64, expectedMethodName string, returnError error, returnOutput []byte) {
	contractMatcher := func(i interface{}) bool {
		input, ok := i.(*services.EthereumCallContractInput)
		return ok &&
			input.EthereumContractAddress == expectedContractAddress &&
			input.EthereumBlockNumber == expectedBlockNumber &&
			input.EthereumFunctionName == expectedMethodName
	}

	outputToReturn := &services.EthereumCallContractOutput{
		EthereumAbiPackedOutput: returnOutput,
	}

	h.crosschainConnectors[protocol.CROSSCHAIN_CONNECTOR_TYPE_ETHEREUM].When("EthereumCallContract", mock.Any, mock.AnyIf(fmt.Sprintf("Contract equals %s, block number equals %d and method equals %s", expectedContractAddress, expectedBlockNumber, expectedMethodName), contractMatcher)).Return(outputToReturn, returnError).Times(1)
}

func (h *harness) verifyEthereumConnectorMethodCalled(t *testing.T) {
	ok, err := h.crosschainConnectors[protocol.CROSSCHAIN_CONNECTOR_TYPE_ETHEREUM].Verify()
	require.True(t, ok, "did not call ethereum connector: %v", err)
}

func (h *harness) expectEthereumConnectorGetTransactionLogs(expectedContractAddress string, expectedEventName string, expectedTxHash string, returnError error, returnOutput []byte, returnBlockNumber uint64, returnTxIndex uint32) {
	contractMatcher := func(i interface{}) bool {
		input, ok := i.(*services.EthereumGetTransactionLogsInput)
		return ok &&
			input.EthereumContractAddress == expectedContractAddress &&
			input.EthereumEventName == expectedEventName &&
			input.EthereumTxhash == expectedTxHash
	}

	outputToReturn := &services.EthereumGetTransactionLogsOutput{
		EthereumAbiPackedOutputs: [][]byte{returnOutput},
		EthereumBlockNumber:      returnBlockNumber,
		EthereumTxindex:          returnTxIndex,
	}

	h.crosschainConnectors[protocol.CROSSCHAIN_CONNECTOR_TYPE_ETHEREUM].When("EthereumGetTransactionLogs", mock.Any, mock.AnyIf(fmt.Sprintf("Contract equals %s, event equals %s and txHash equals %s", expectedContractAddress, expectedEventName, expectedTxHash), contractMatcher)).Return(outputToReturn, returnError).Times(1)
}

func (h *harness) expectEthereumConnectorGetBlockNumber(returnError error, returnBlockNumber uint64) {
	contractMatcher := func(i interface{}) bool {
		_, ok := i.(*services.EthereumGetBlockNumberInput)
		return ok
	}

	outputToReturn := &services.EthereumGetBlockNumberOutput{
		EthereumBlockNumber: returnBlockNumber,
	}

	h.crosschainConnectors[protocol.CROSSCHAIN_CONNECTOR_TYPE_ETHEREUM].When("EthereumGetBlockNumber", mock.Any, mock.AnyIf("any input", contractMatcher)).Return(outputToReturn, returnError).Times(1)
}

func (h *harness) expectStateStorageBlockHeightRequested(returnValue primitives.BlockHeight) {
	outputToReturn := &services.GetStateStorageBlockHeightOutput{
		LastCommittedBlockHeight:    returnValue,
		LastCommittedBlockTimestamp: 1234,
	}

	h.stateStorage.When("GetStateStorageBlockHeight", mock.Any, mock.Any).Return(outputToReturn, nil).Times(1)
}

func (h *harness) expectStateStorageBlockHeightAndTimestampRequested(returnHeight primitives.BlockHeight, returnTimestamp primitives.TimestampNano) {
	outputToReturn := &services.GetStateStorageBlockHeightOutput{
		LastCommittedBlockHeight:    returnHeight,
		LastCommittedBlockTimestamp: returnTimestamp,
	}

	h.stateStorage.When("GetStateStorageBlockHeight", mock.Any, mock.Any).Return(outputToReturn, nil).Times(1)
}

func (h *harness) verifyStateStorageBlockHeightRequested(t *testing.T) {
	ok, err := h.stateStorage.Verify()
	require.True(t, ok, "did not read from state storage: %v", err)
}

func (h *harness) expectStateStorageRead(expectedHeight primitives.BlockHeight, expectedContractName primitives.ContractName, expectedKey []byte, returnValue []byte) {
	stateReadMatcher := func(i interface{}) bool {
		input, ok := i.(*services.ReadKeysInput)
		return ok &&
			input.BlockHeight == expectedHeight &&
			input.ContractName == expectedContractName &&
			len(input.Keys) == 1 &&
			bytes.Equal(input.Keys[0], expectedKey)
	}

	outputToReturn := &services.ReadKeysOutput{
		StateRecords: []*protocol.StateRecord{(&protocol.StateRecordBuilder{
			Key:   expectedKey,
			Value: returnValue,
		}).Build()},
	}

	h.stateStorage.When("ReadKeys", mock.Any, mock.AnyIf(fmt.Sprintf("ReadKeys height equals %s and key equals %x", expectedHeight, expectedKey), stateReadMatcher)).Return(outputToReturn, nil).Times(1)
}

func (h *harness) verifyStateStorageRead(t *testing.T) {
	ok, err := h.stateStorage.Verify()
	require.True(t, ok, "state storage read was not expected: %v", err)
}

func (h *harness) expectStateStorageNotRead() {
	h.stateStorage.When("ReadKeys", mock.Any, mock.Any).Return(&services.ReadKeysOutput{
		StateRecords: []*protocol.StateRecord{
			(&protocol.StateRecordBuilder{
				Key:   []byte{0x01},
				Value: []byte{0x02},
			}).Build(),
		},
	}, nil).Times(0)
}
