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

func TestProcessQuery_WhenContractNotDeployed(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)

		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, errors.New("not deployed"), uint32(0))

		h.expectStateStorageBlockHeightRequested(12)
		h.expectNativeContractMethodNotCalled("Contract1", "method1")

		result, outputArgs, refHeight, _, err := h.processQuery(ctx, "Contract1", "method1")
		require.Error(t, err, "process query should fail")
		require.Equal(t, protocol.EXECUTION_RESULT_ERROR_CONTRACT_NOT_DEPLOYED, result, "process query should return not deployed")
		require.Equal(t, []byte{}, outputArgs, "process query should return matching output args")
		require.EqualValues(t, 12, refHeight)

		h.verifySystemContractCalled(t)
		h.verifyStateStorageBlockHeightRequested(t)
		h.verifyNativeContractMethodCalled(t)
	})
}

func TestProcessTransactionSet_WhenContractNotDeployedAndNotPreBuiltNativeContract(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)

		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, errors.New("not deployed"), uint32(0))
		h.expectNativeContractInfoRequested("Contract1", errors.New("not found"))

		h.expectNativeContractMethodNotCalled("Contract1", "method1")

		results, outputArgs, _, _ := h.processTransactionSet(ctx, []*contractAndMethod{
			{"Contract1", "method1"},
		})
		require.Equal(t, results, []protocol.ExecutionResult{
			protocol.EXECUTION_RESULT_ERROR_CONTRACT_NOT_DEPLOYED,
		}, "processTransactionSet returned receipts should match")
		require.Equal(t, outputArgs, [][]byte{
			{},
		}, "processTransactionSet returned output args should match")

		h.verifySystemContractCalled(t)
		h.verifyNativeContractInfoRequested(t)
		h.verifyNativeContractMethodCalled(t)
	})
}

func TestSdkService_CallMethodWhenContractNotDeployedAndNotPreBuiltNativeContract(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)

		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, errors.New("not deployed"), uint32(0))
		h.expectNativeContractInfoRequested("Contract2", errors.New("not found"))

		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("CallMethod on non deployed contract")
			_, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_SERVICE, "callMethod", "Contract2", "method1", builders.ArgumentsArray().Raw())
			require.Error(t, err, "handleSdkCall should fail")
			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})
		h.expectNativeContractMethodNotCalled("Contract2", "method1")

		h.processTransactionSet(ctx, []*contractAndMethod{
			{"Contract1", "method1"},
		})

		h.verifySystemContractCalled(t)
		h.verifyNativeContractInfoRequested(t)
		h.verifyNativeContractMethodCalled(t)
	})
}

func TestAutoDeployPreBuiltNativeContractDuringProcessTransactionSet(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)

		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, errors.New("not deployed"), uint32(0))
		h.expectNativeContractInfoRequested("Contract1", nil)

		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_DEPLOY_SERVICE, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE))
		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})

		results, outputArgs, _, _ := h.processTransactionSet(ctx, []*contractAndMethod{
			{"Contract1", "method1"},
		})
		require.Equal(t, results, []protocol.ExecutionResult{
			protocol.EXECUTION_RESULT_SUCCESS,
		}, "processTransactionSet returned receipts should match")
		require.Equal(t, outputArgs, [][]byte{
			{},
		}, "processTransactionSet returned output args should match")

		h.verifySystemContractCalled(t)
		h.verifyNativeContractInfoRequested(t)
		h.verifyNativeContractMethodCalled(t)
	})
}

func TestFailingAutoDeployPreBuiltNativeContractDuringProcessTransactionSet(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)

		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, errors.New("not deployed"), uint32(0))
		h.expectNativeContractInfoRequested("Contract1", nil)

		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_DEPLOY_SERVICE, errors.New("deploy error"), uint32(0))
		h.expectNativeContractMethodNotCalled("Contract1", "method1")

		results, outputArgs, _, _ := h.processTransactionSet(ctx, []*contractAndMethod{
			{"Contract1", "method1"},
		})
		require.Equal(t, results, []protocol.ExecutionResult{
			protocol.EXECUTION_RESULT_ERROR_CONTRACT_NOT_DEPLOYED,
		}, "processTransactionSet returned receipts should match")
		require.Equal(t, outputArgs, [][]byte{
			{},
		}, "processTransactionSet returned output args should match")

		h.verifySystemContractCalled(t)
		h.verifyNativeContractInfoRequested(t)
		h.verifyNativeContractMethodCalled(t)
	})
}
