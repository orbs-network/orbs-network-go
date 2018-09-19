package test

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProcessTransactionSetSuccess(t *testing.T) {
	h := newHarness()
	h.expectSystemContractCalled(deployments.CONTRACT.Name, deployments.METHOD_GET_INFO.Name, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

	h.expectNativeContractMethodCalled("Contract1", "method1", func(contextId primitives.ExecutionContextId, inputArgs *protocol.MethodArgumentArray) (protocol.ExecutionResult, *protocol.MethodArgumentArray, error) {
		t.Log("Transaction 1: successful")
		return protocol.EXECUTION_RESULT_SUCCESS, builders.MethodArgumentsArray(), nil
	})
	h.expectNativeContractMethodCalled("Contract1", "method2", func(contextId primitives.ExecutionContextId, inputArgs *protocol.MethodArgumentArray) (protocol.ExecutionResult, *protocol.MethodArgumentArray, error) {
		t.Log("Transaction 2: successful")
		return protocol.EXECUTION_RESULT_SUCCESS, builders.MethodArgumentsArray(uint32(17), "hello", []byte{0x01, 0x02}), nil
	})

	results, outputArgs, _ := h.processTransactionSet([]*contractAndMethod{
		{"Contract1", "method1"},
		{"Contract1", "method2"},
	})
	require.Equal(t, results, []protocol.ExecutionResult{
		protocol.EXECUTION_RESULT_SUCCESS,
		protocol.EXECUTION_RESULT_SUCCESS,
	}, "processTransactionSet returned receipts should match")
	require.Equal(t, outputArgs, [][]byte{
		{},
		builders.MethodArgumentsOpaqueEncode(uint32(17), "hello", []byte{0x01, 0x02}),
	}, "processTransactionSet returned output args should match")

	h.verifySystemContractCalled(t)
	h.verifyNativeContractMethodCalled(t)
}

func TestProcessTransactionSetWithErrors(t *testing.T) {
	h := newHarness()
	h.expectSystemContractCalled(deployments.CONTRACT.Name, deployments.METHOD_GET_INFO.Name, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

	h.expectNativeContractMethodCalled("Contract1", "method1", func(contextId primitives.ExecutionContextId, inputArgs *protocol.MethodArgumentArray) (protocol.ExecutionResult, *protocol.MethodArgumentArray, error) {
		t.Log("Transaction 1: failed (contract error)")
		return protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT, builders.MethodArgumentsArray(), errors.New("contract error")
	})
	h.expectNativeContractMethodCalled("Contract1", "method2", func(contextId primitives.ExecutionContextId, inputArgs *protocol.MethodArgumentArray) (protocol.ExecutionResult, *protocol.MethodArgumentArray, error) {
		t.Log("Transaction 2: failed (unexpected error)")
		return protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, builders.MethodArgumentsArray(), errors.New("unexpected error")
	})

	results, outputArgs, _ := h.processTransactionSet([]*contractAndMethod{
		{"Contract1", "method1"},
		{"Contract1", "method2"},
	})
	require.Equal(t, results, []protocol.ExecutionResult{
		protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT,
		protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
	}, "processTransactionSet returned receipts should match")
	require.Equal(t, outputArgs, [][]byte{
		{},
		{},
	}, "processTransactionSet returned output args should match")

	h.verifySystemContractCalled(t)
	h.verifyNativeContractMethodCalled(t)
}
