package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRunLocalMethod_Success(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness()
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

		h.expectStateStorageBlockHeightRequested(12)
		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.MethodArgumentArray) (protocol.ExecutionResult, *protocol.MethodArgumentArray, error) {
			return protocol.EXECUTION_RESULT_SUCCESS, builders.MethodArgumentsArray(uint32(17), "hello", []byte{0x01, 0x02}), nil
		})

		result, outputArgs, refHeight, err := h.runLocalMethod(ctx, "Contract1", "method1")
		require.NoError(t, err, "run local method should not fail")
		require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, result, "run local method should return successful result")
		require.Equal(t, builders.MethodArgumentsOpaqueEncode(uint32(17), "hello", []byte{0x01, 0x02}), outputArgs, "run local method should return matching output args")
		require.EqualValues(t, 12, refHeight)

		h.verifySystemContractCalled(t)
		h.verifyStateStorageBlockHeightRequested(t)
		h.verifyNativeContractMethodCalled(t)
	})
}

func TestRunLocalMethod_ContractError(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness()
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

		h.expectStateStorageBlockHeightRequested(12)
		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.MethodArgumentArray) (protocol.ExecutionResult, *protocol.MethodArgumentArray, error) {
			return protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT, builders.MethodArgumentsArray(), errors.New("contract error")
		})

		result, outputArgs, refHeight, err := h.runLocalMethod(ctx, "Contract1", "method1")
		require.Error(t, err, "run local method should fail")
		require.Equal(t, protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT, result, "run local method should return contract error")
		require.Equal(t, []byte{}, outputArgs, "run local method should return matching output args")
		require.EqualValues(t, 12, refHeight)

		h.verifySystemContractCalled(t)
		h.verifyStateStorageBlockHeightRequested(t)
		h.verifyNativeContractMethodCalled(t)
	})
}

func TestRunLocalMethod_UnexpectedError(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness()
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

		h.expectStateStorageBlockHeightRequested(12)
		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.MethodArgumentArray) (protocol.ExecutionResult, *protocol.MethodArgumentArray, error) {
			return protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, builders.MethodArgumentsArray(), errors.New("unexpected error")
		})

		result, outputArgs, refHeight, err := h.runLocalMethod(ctx, "Contract1", "method1")
		require.Error(t, err, "run local method should fail")
		require.Equal(t, protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, result, "run local method should return unexpected error")
		require.Equal(t, []byte{}, outputArgs, "run local method should return matching output args")
		require.EqualValues(t, 12, refHeight)

		h.verifySystemContractCalled(t)
		h.verifyStateStorageBlockHeightRequested(t)
		h.verifyNativeContractMethodCalled(t)
	})
}
