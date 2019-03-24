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
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSdkEnv_GetBlockDetails_InTransaction(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

		const currentBlockHeight = primitives.BlockHeight(12)
		const currentBlockTimestamp = primitives.TimestampNano(0x777)

		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("getBlockHeight")
			res, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_ENV, "getBlockHeight")
			require.NoError(t, err, "handleSdkCall should not fail")
			require.Equal(t, uint64(currentBlockHeight), res[0].Uint64Value(), "handleSdkCall result should be equal")

			t.Log("getBlockTimestamp")
			res, err = h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_ENV, "getBlockTimestamp")
			require.NoError(t, err, "handleSdkCall should not fail")
			require.Equal(t, uint64(currentBlockTimestamp), res[0].Uint64Value(), "handleSdkCall result should be equal")

			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})

		h.processTransactionSetAtHeightAndTimestamp(ctx, currentBlockHeight, currentBlockTimestamp, []*contractAndMethod{
			{"Contract1", "method1"},
		})

		h.verifySystemContractCalled(t)
		h.verifyNativeContractMethodCalled(t)
	})
}

func TestSdkEnv_GetBlockDetails_InCallMethod(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

		const lastCommittedBlockHeight = primitives.BlockHeight(12)
		const lastCommittedBlockTimestamp = primitives.TimestampNano(0x777)

		h.expectStateStorageBlockHeightAndTimestampRequested(lastCommittedBlockHeight, lastCommittedBlockTimestamp)
		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("getBlockHeight")
			res, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_ENV, "getBlockHeight")
			require.NoError(t, err, "handleSdkCall should not fail")
			require.Equal(t, uint64(lastCommittedBlockHeight), res[0].Uint64Value(), "handleSdkCall result should be equal")

			t.Log("getBlockTimestamp")
			res, err = h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_ENV, "getBlockTimestamp")
			require.NoError(t, err, "handleSdkCall should not fail")
			require.Equal(t, uint64(lastCommittedBlockTimestamp), res[0].Uint64Value(), "handleSdkCall result should be equal")

			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})

		h.processQuery(ctx, "Contract1", "method1")

		h.verifySystemContractCalled(t)
		h.verifyStateStorageBlockHeightRequested(t)
		h.verifyNativeContractMethodCalled(t)
	})
}
