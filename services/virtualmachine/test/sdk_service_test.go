package test

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSdkServiceCallMethodFailingCall(t *testing.T) {
	h := newHarness()
	h.expectSystemContractCalled(deployments.CONTRACT.Name, deployments.METHOD_GET_INFO.Name, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

	h.expectNativeContractMethodCalled("Contract1", "method1", func(contextId primitives.ExecutionContextId, inputArgs *protocol.MethodArgumentArray) (protocol.ExecutionResult, *protocol.MethodArgumentArray, error) {
		t.Log("CallMethod on failing contract")
		_, err := h.handleSdkCall(contextId, native.SDK_OPERATION_NAME_SERVICE, "callMethod", "FailingContract", "method1")
		require.Error(t, err, "handleSdkCall should fail")
		return protocol.EXECUTION_RESULT_SUCCESS, builders.MethodArgumentsArray(), nil
	})
	h.expectNativeContractMethodCalled("FailingContract", "method1", func(contextId primitives.ExecutionContextId, inputArgs *protocol.MethodArgumentArray) (protocol.ExecutionResult, *protocol.MethodArgumentArray, error) {
		return protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, builders.MethodArgumentsArray(), errors.New("call error")
	})

	h.processTransactionSet([]*contractAndMethod{
		{"Contract1", "method1"},
	})

	h.verifySystemContractCalled(t)
	h.verifyNativeContractMethodCalled(t)
}

func TestSdkServiceCallMethodMaintainsAddressSpaceUnderSameContract(t *testing.T) {
	h := newHarness()
	h.expectSystemContractCalled(deployments.CONTRACT.Name, deployments.METHOD_GET_INFO.Name, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

	h.expectNativeContractMethodCalled("Contract1", "method1", func(contextId primitives.ExecutionContextId, inputArgs *protocol.MethodArgumentArray) (protocol.ExecutionResult, *protocol.MethodArgumentArray, error) {
		t.Log("Write to key in first contract")
		_, err := h.handleSdkCall(contextId, native.SDK_OPERATION_NAME_STATE, "write", []byte{0x01}, []byte{0x02, 0x03})
		require.NoError(t, err, "handleSdkCall should succeed")

		t.Log("CallMethod on a the same contract")
		_, err = h.handleSdkCall(contextId, native.SDK_OPERATION_NAME_SERVICE, "callMethod", "Contract1", "method2")
		require.NoError(t, err, "handleSdkCall should succeed")

		return protocol.EXECUTION_RESULT_SUCCESS, builders.MethodArgumentsArray(), nil
	})
	h.expectNativeContractMethodCalled("Contract1", "method2", func(contextId primitives.ExecutionContextId, inputArgs *protocol.MethodArgumentArray) (protocol.ExecutionResult, *protocol.MethodArgumentArray, error) {
		t.Log("Read the same key in the first contract")
		res, err := h.handleSdkCall(contextId, native.SDK_OPERATION_NAME_STATE, "read", []byte{0x01})
		require.NoError(t, err, "handleSdkCall should not fail")
		require.Equal(t, []byte{0x02, 0x03}, res[0].BytesValue(), "handleSdkCall result should be equal")
		return protocol.EXECUTION_RESULT_SUCCESS, builders.MethodArgumentsArray(), nil
	})
	h.expectStateStorageNotRead()

	h.processTransactionSet([]*contractAndMethod{
		{"Contract1", "method1"},
	})

	h.verifySystemContractCalled(t)
	h.verifyNativeContractMethodCalled(t)
	h.verifyStateStorageRead(t)
}

func TestSdkServiceCallMethodChangesAddressSpaceBetweenContracts(t *testing.T) {
	h := newHarness()
	h.expectSystemContractCalled(deployments.CONTRACT.Name, deployments.METHOD_GET_INFO.Name, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

	h.expectNativeContractMethodCalled("Contract1", "method1", func(contextId primitives.ExecutionContextId, inputArgs *protocol.MethodArgumentArray) (protocol.ExecutionResult, *protocol.MethodArgumentArray, error) {
		t.Log("Write to key in first contract")
		_, err := h.handleSdkCall(contextId, native.SDK_OPERATION_NAME_STATE, "write", []byte{0x01}, []byte{0x02, 0x03})
		require.NoError(t, err, "handleSdkCall should succeed")

		t.Log("CallMethod on a different contract")
		_, err = h.handleSdkCall(contextId, native.SDK_OPERATION_NAME_SERVICE, "callMethod", "Contract2", "method1")
		require.NoError(t, err, "handleSdkCall should succeed")

		t.Log("Read the same key in the first contract after the call")
		res, err := h.handleSdkCall(contextId, native.SDK_OPERATION_NAME_STATE, "read", []byte{0x01})
		require.NoError(t, err, "handleSdkCall should not fail")
		require.Equal(t, []byte{0x02, 0x03}, res[0].BytesValue(), "handleSdkCall result should be equal")

		return protocol.EXECUTION_RESULT_SUCCESS, builders.MethodArgumentsArray(), nil
	})
	h.expectNativeContractMethodCalled("Contract2", "method1", func(contextId primitives.ExecutionContextId, inputArgs *protocol.MethodArgumentArray) (protocol.ExecutionResult, *protocol.MethodArgumentArray, error) {
		t.Log("Read the same key in the second contract")
		res, err := h.handleSdkCall(contextId, native.SDK_OPERATION_NAME_STATE, "read", []byte{0x01})
		require.NoError(t, err, "handleSdkCall should not fail")
		require.Equal(t, []byte{0x04, 0x05, 0x06}, res[0].BytesValue(), "handleSdkCall result should be equal")
		return protocol.EXECUTION_RESULT_SUCCESS, builders.MethodArgumentsArray(), nil
	})
	h.expectStateStorageRead(11, "Contract2", []byte{0x01}, []byte{0x04, 0x05, 0x06})

	h.processTransactionSet([]*contractAndMethod{
		{"Contract1", "method1"},
	})

	h.verifySystemContractCalled(t)
	h.verifyNativeContractMethodCalled(t)
	h.verifyStateStorageRead(t)
}

func TestSdkServiceCallMethodWithSystemPermissions(t *testing.T) {
	h := newHarness()
	h.expectSystemContractCalled(deployments.CONTRACT.Name, deployments.METHOD_GET_INFO.Name, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

	h.expectNativeContractMethodCalled("Contract1", "method1", func(contextId primitives.ExecutionContextId, inputArgs *protocol.MethodArgumentArray) (protocol.ExecutionResult, *protocol.MethodArgumentArray, error) {
		t.Log("CallMethod on a different contract with system permissions")
		_, err := h.handleSdkCallWithSystemPermissions(contextId, native.SDK_OPERATION_NAME_SERVICE, "callMethod", "Contract2", "method1")
		require.NoError(t, err, "handleSdkCall should succeed")

		return protocol.EXECUTION_RESULT_SUCCESS, builders.MethodArgumentsArray(), nil
	})
	h.expectNativeContractMethodCalledWithSystemPermissions("Contract2", "method1", func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, *protocol.MethodArgumentArray, error) {
		return protocol.EXECUTION_RESULT_SUCCESS, builders.MethodArgumentsArray(), nil
	})

	h.processTransactionSet([]*contractAndMethod{
		{"Contract1", "method1"},
	})

	h.verifySystemContractCalled(t)
	h.verifyNativeContractMethodCalled(t)
}
