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

func TestRunLocalMethod_WhenContractNotDeployed(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness()

		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT.Name, deployments_systemcontract.METHOD_GET_INFO.Name, errors.New("not deployed"), uint32(0))

		h.expectStateStorageBlockHeightRequested(12)
		h.expectNativeContractMethodNotCalled("Contract1", "method1")

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

func TestProcessTransactionSet_WhenContractNotDeployedAndNotPreBuiltNativeContract(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness()

		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT.Name, deployments_systemcontract.METHOD_GET_INFO.Name, errors.New("not deployed"), uint32(0))
		h.expectNativeContractInfoRequested("Contract1", errors.New("not found"))

		h.expectNativeContractMethodNotCalled("Contract1", "method1")

		results, outputArgs, _ := h.processTransactionSet(ctx, []*contractAndMethod{
			{"Contract1", "method1"},
		})
		require.Equal(t, results, []protocol.ExecutionResult{
			protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
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
		h := newHarness()

		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT.Name, deployments_systemcontract.METHOD_GET_INFO.Name, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT.Name, deployments_systemcontract.METHOD_GET_INFO.Name, errors.New("not deployed"), uint32(0))
		h.expectNativeContractInfoRequested("Contract2", errors.New("not found"))

		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.MethodArgumentArray) (protocol.ExecutionResult, *protocol.MethodArgumentArray, error) {
			t.Log("CallMethod on non deployed contract")
			_, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_SERVICE, "callMethod", "Contract2", "method1", builders.MethodArgumentsArray().Raw())
			require.Error(t, err, "handleSdkCall should fail")
			return protocol.EXECUTION_RESULT_SUCCESS, builders.MethodArgumentsArray(), nil
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
		h := newHarness()

		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT.Name, deployments_systemcontract.METHOD_GET_INFO.Name, errors.New("not deployed"), uint32(0))
		h.expectNativeContractInfoRequested("Contract1", nil)

		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT.Name, deployments_systemcontract.METHOD_DEPLOY_SERVICE.Name, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE))
		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.MethodArgumentArray) (protocol.ExecutionResult, *protocol.MethodArgumentArray, error) {
			return protocol.EXECUTION_RESULT_SUCCESS, builders.MethodArgumentsArray(), nil
		})

		results, outputArgs, _ := h.processTransactionSet(ctx, []*contractAndMethod{
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
		h := newHarness()

		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT.Name, deployments_systemcontract.METHOD_GET_INFO.Name, errors.New("not deployed"), uint32(0))
		h.expectNativeContractInfoRequested("Contract1", nil)

		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT.Name, deployments_systemcontract.METHOD_DEPLOY_SERVICE.Name, errors.New("deploy error"), uint32(0))
		h.expectNativeContractMethodNotCalled("Contract1", "method1")

		results, outputArgs, _ := h.processTransactionSet(ctx, []*contractAndMethod{
			{"Contract1", "method1"},
		})
		require.Equal(t, results, []protocol.ExecutionResult{
			protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		}, "processTransactionSet returned receipts should match")
		require.Equal(t, outputArgs, [][]byte{
			{},
		}, "processTransactionSet returned output args should match")

		h.verifySystemContractCalled(t)
		h.verifyNativeContractInfoRequested(t)
		h.verifyNativeContractMethodCalled(t)
	})
}
