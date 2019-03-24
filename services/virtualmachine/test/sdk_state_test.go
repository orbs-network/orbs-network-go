// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSdkState_ReadWithoutContext(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)

		_, err := h.handleSdkCall(ctx, EXAMPLE_CONTEXT_ID, native.SDK_OPERATION_NAME_STATE, "read", []byte{0x01})
		require.Error(t, err, "handleSdkCall should fail")
	})
}

func TestSdkState_ReadWithLocalMethodReadOnlyAccess(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

		h.expectStateStorageBlockHeightRequested(12)
		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("First read should reach state storage")
			res, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "read", []byte{0x01})
			require.NoError(t, err, "handleSdkCall should not fail")
			require.Equal(t, []byte{0x02}, res[0].BytesValue(), "handleSdkCall result should be equal")

			t.Log("Second read should be from cache")
			res, err = h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "read", []byte{0x01})
			require.NoError(t, err, "handleSdkCall should not fail")
			require.Equal(t, []byte{0x02}, res[0].BytesValue(), "handleSdkCall result should be equal")

			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})
		h.expectStateStorageRead(12, "Contract1", []byte{0x01}, []byte{0x02})

		h.processQuery(ctx, "Contract1", "method1")

		h.verifySystemContractCalled(t)
		h.verifyStateStorageBlockHeightRequested(t)
		h.verifyNativeContractMethodCalled(t)
		h.verifyStateStorageRead(t)
	})
}

func TestSdkState_WriteWithLocalMethodReadOnlyAccess(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

		h.expectStateStorageBlockHeightRequested(12)
		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Attempt to write without proper access")
			_, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "write", []byte{0x01}, []byte{0x02})
			require.Error(t, err, "handleSdkCall should fail")
			return protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, builders.ArgumentsArray(), errors.New("unexpected error")
		})

		h.processQuery(ctx, "Contract1", "method1")

		h.verifySystemContractCalled(t)
		h.verifyStateStorageBlockHeightRequested(t)
		h.verifyNativeContractMethodCalled(t)
	})
}

func TestSdkState_ReadWithTransactionSetReadWriteAccess(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("First read should reach state storage")
			res, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "read", []byte{0x01})
			require.NoError(t, err, "handleSdkCall should not fail")
			require.Equal(t, []byte{0x02}, res[0].BytesValue(), "handleSdkCall result should be equal")

			t.Log("Second read should be from cache")
			res, err = h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "read", []byte{0x01})
			require.NoError(t, err, "handleSdkCall should not fail")
			require.Equal(t, []byte{0x02}, res[0].BytesValue(), "handleSdkCall result should be equal")

			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})
		h.expectStateStorageRead(11, "Contract1", []byte{0x01}, []byte{0x02})

		_, _, sd, _ := h.processTransactionSet(ctx, []*contractAndMethod{
			{"Contract1", "method1"},
		})
		require.ElementsMatch(t, sd["Contract1"], []*keyValuePair{}, "processTransactionSet returned contract state diffs should be empty")

		h.verifySystemContractCalled(t)
		h.verifyNativeContractMethodCalled(t)
		h.verifyStateStorageRead(t)
	})
}

func TestSdkState_WriteWithTransactionSetReadWriteAccess(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Transaction 1: first write should change in transient state")
			_, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "write", []byte{0x01}, []byte{0x02})
			require.NoError(t, err, "handleSdkCall should succeed")

			t.Log("Transaction 1: second write should replace in transient state")
			_, err = h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "write", []byte{0x01}, []byte{0x03, 0x04})
			require.NoError(t, err, "handleSdkCall should succeed")

			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})
		h.expectNativeContractMethodCalled("Contract1", "method2", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Transaction 2: first write should replace in transient state")
			_, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "write", []byte{0x01}, []byte{0x05, 0x06})
			require.NoError(t, err, "handleSdkCall should succeed")

			t.Log("Transaction 2: read should return last successful write")
			res, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "read", []byte{0x01})
			require.NoError(t, err, "handleSdkCall should not fail")
			require.Equal(t, []byte{0x05, 0x06}, res[0].BytesValue(), "handleSdkCall result should be equal")

			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})
		h.expectStateStorageNotRead()

		_, _, sd, _ := h.processTransactionSet(ctx, []*contractAndMethod{
			{"Contract1", "method1"},
			{"Contract1", "method2"},
		})
		require.ElementsMatch(t, sd["Contract1"], []*keyValuePair{
			{[]byte{0x01}, []byte{0x05, 0x06}},
		}, "processTransactionSet returned contract state diffs should match")

		h.verifySystemContractCalled(t)
		h.verifyNativeContractMethodCalled(t)
		h.verifyStateStorageRead(t)
	})
}

func TestSdkState_WriteOfDifferentContractsDoNotOverrideEachOther(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Transaction 1: write to key in first contract")
			_, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "write", []byte{0x01}, []byte{0x02})
			require.NoError(t, err, "handleSdkCall should succeed")
			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})
		h.expectNativeContractMethodCalled("Contract2", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Transaction 2: write to same key in second contract")
			_, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "write", []byte{0x01}, []byte{0x03, 0x04})
			require.NoError(t, err, "handleSdkCall should succeed")
			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})
		h.expectNativeContractMethodCalled("Contract1", "method2", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Transaction 3: read from first contract")
			res, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "read", []byte{0x01})
			require.NoError(t, err, "handleSdkCall should not fail")
			require.Equal(t, []byte{0x02}, res[0].BytesValue(), "handleSdkCall result should be equal")
			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})
		h.expectNativeContractMethodCalled("Contract2", "method2", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Transaction 4: read from second contract")
			res, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "read", []byte{0x01})
			require.NoError(t, err, "handleSdkCall should not fail")
			require.Equal(t, []byte{0x03, 0x04}, res[0].BytesValue(), "handleSdkCall result should be equal")
			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})
		h.expectStateStorageNotRead()

		_, _, sd, _ := h.processTransactionSet(ctx, []*contractAndMethod{
			{"Contract1", "method1"},
			{"Contract2", "method1"},
			{"Contract1", "method2"},
			{"Contract2", "method2"},
		})
		require.ElementsMatch(t, sd["Contract1"], []*keyValuePair{
			{[]byte{0x01}, []byte{0x02}},
		}, "processTransactionSet returned contract state diffs should match")
		require.ElementsMatch(t, sd["Contract2"], []*keyValuePair{
			{[]byte{0x01}, []byte{0x03, 0x04}},
		}, "processTransactionSet returned contract state diffs should match")

		h.verifySystemContractCalled(t)
		h.verifyNativeContractMethodCalled(t)
		h.verifyStateStorageRead(t)
	})
}

func TestSdkState_WriteIgnoredWithTransactionSetHavingFailedTransactions(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Transaction 1 (successful): first write should change in transient state")
			_, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "write", []byte{0x01}, []byte{0x02})
			require.NoError(t, err, "handleSdkCall should succeed")
			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})
		h.expectNativeContractMethodCalled("Contract1", "method2", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Transaction 2 (failed): write should be ignored")
			_, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "write", []byte{0x01}, []byte{0x03, 0x04})
			require.NoError(t, err, "handleSdkCall should succeed")

			t.Log("Transaction 2 (failed): read the ignored write should return it")
			res, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "read", []byte{0x01})
			require.NoError(t, err, "handleSdkCall should not fail")
			require.Equal(t, []byte{0x03, 0x04}, res[0].BytesValue(), "handleSdkCall result should be equal")

			return protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT, builders.ArgumentsArray(), errors.New("contract error")
		})
		h.expectNativeContractMethodCalled("Contract1", "method3", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Transaction 3 (successful): read should return last successful write")
			res, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "read", []byte{0x01})
			require.NoError(t, err, "handleSdkCall should not fail")
			require.Equal(t, []byte{0x02}, res[0].BytesValue(), "handleSdkCall result should be equal")
			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})
		h.expectStateStorageNotRead()

		_, _, sd, _ := h.processTransactionSet(ctx, []*contractAndMethod{
			{"Contract1", "method1"},
			{"Contract1", "method2"},
			{"Contract1", "method3"},
		})
		require.ElementsMatch(t, sd["Contract1"], []*keyValuePair{
			{[]byte{0x01}, []byte{0x02}},
		}, "processTransactionSet returned contract state diffs should match")

		h.verifySystemContractCalled(t)
		h.verifyNativeContractMethodCalled(t)
		h.verifyStateStorageRead(t)
	})
}
