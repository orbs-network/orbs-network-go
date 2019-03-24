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

func TestProcessTransactionSet_WhenContractNotDeployedAndIsPreBuiltNativeContract_ButSafeFromDoubleDeploy(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)

		// first transaction should deploy to transient state
		h.expectPreBuiltContractNotToBeDeployed()
		h.expectDeployToWriteDeploymentDataToState(t, ctx)
		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})

		// second transaction should read deployment data from transient state
		h.expectContractToBeDeployedByReadingDeploymentDataFromState(t, ctx)
		h.expectStateStorageNotRead() // we expect the read to come from transient state, not the state storage service
		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})

		results, _, _, _ := h.processTransactionSet(ctx, []*contractAndMethod{
			{"Contract1", "method1"},
			{"Contract1", "method1"},
		}, DEPLOYMENT_CONTRACT)
		require.Equal(t, results, []protocol.ExecutionResult{
			protocol.EXECUTION_RESULT_SUCCESS,
			protocol.EXECUTION_RESULT_SUCCESS,
		}, "processTransactionSet returned receipts should match")

		h.verifyNativeContractMethodCalled(t)
		h.verifyNativeContractInfoRequested(t)
		h.verifyStateStorageRead(t)
	})
}

var DEPLOYMENT_CONTRACT = primitives.ContractName(deployments_systemcontract.CONTRACT_NAME)
var DEPLOYMENT_GET_INFO_METHOD = primitives.MethodName(deployments_systemcontract.METHOD_GET_INFO)
var DEPLOYMENT_DEPLOY_METHOD = primitives.MethodName(deployments_systemcontract.METHOD_DEPLOY_SERVICE)
var DEPLOYMENT_DATA_STATE_KEY_NAME = []byte{0x01}  // some value to represent it
var DEPLOYMENT_DATA_STATE_KEY_VALUE = []byte{0x02} // some value to represent it

func (h *harness) expectPreBuiltContractNotToBeDeployed() {
	h.expectNativeContractMethodCalled(DEPLOYMENT_CONTRACT, DEPLOYMENT_GET_INFO_METHOD, func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
		return protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT, builders.ArgumentsArray(), errors.New("not deployed")
	})
	h.expectNativeContractInfoRequested("Contract1", nil)
}

func (h *harness) expectDeployToWriteDeploymentDataToState(t *testing.T, ctx context.Context) {
	h.expectNativeContractMethodCalled(DEPLOYMENT_CONTRACT, DEPLOYMENT_DEPLOY_METHOD, func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
		t.Log("Transaction: deploy writes deployment data to state")
		_, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "write", DEPLOYMENT_DATA_STATE_KEY_NAME, DEPLOYMENT_DATA_STATE_KEY_VALUE)
		require.NoError(t, err, "handleSdkCall should succeed")
		return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(uint32(protocol.PROCESSOR_TYPE_NATIVE)), nil
	})
}

func (h *harness) expectContractToBeDeployedByReadingDeploymentDataFromState(t *testing.T, ctx context.Context) {
	h.expectNativeContractMethodCalled(DEPLOYMENT_CONTRACT, DEPLOYMENT_GET_INFO_METHOD, func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
		t.Log("Transaction: read should return the deployment data")
		res, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "read", DEPLOYMENT_DATA_STATE_KEY_NAME)
		require.NoError(t, err, "handleSdkCall should not fail")
		require.Equal(t, DEPLOYMENT_DATA_STATE_KEY_VALUE, res[0].BytesValue(), "handleSdkCall result should be equal")
		return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(uint32(protocol.PROCESSOR_TYPE_NATIVE)), nil
	})
}
