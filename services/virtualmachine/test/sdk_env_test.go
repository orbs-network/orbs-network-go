// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/crypto-lib-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-network-go/services/processor/sdk"
	"github.com/orbs-network/orbs-network-go/test/builders"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSdkEnv_GetBlockDetails_InTransaction(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {

			h := newHarness(parent.Logger)
			h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

			const currentBlockHeight = primitives.BlockHeight(12)
			const currentBlockTimestamp = primitives.TimestampNano(0x777)
			currentBlockProposer := hash.Make32BytesWithFirstByte(5)

			h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
				t.Log("getBlockHeight")
				res, err := h.handleSdkCall(ctx, executionContextId, sdk.SDK_OPERATION_NAME_ENV, "getBlockHeight")
				require.NoError(t, err, "handleSdkCall should not fail")
				require.Equal(t, uint64(currentBlockHeight), res[0].Uint64Value(), "handleSdkCall result should be equal")

				t.Log("getBlockTimestamp")
				res, err = h.handleSdkCall(ctx, executionContextId, sdk.SDK_OPERATION_NAME_ENV, "getBlockTimestamp")
				require.NoError(t, err, "handleSdkCall should not fail")
				require.Equal(t, uint64(currentBlockTimestamp), res[0].Uint64Value(), "handleSdkCall result should be equal")

				t.Log("getBlockProposerAddress")
				res, err = h.handleSdkCall(ctx, executionContextId, sdk.SDK_OPERATION_NAME_ENV, "getBlockProposerAddress")
				require.Error(t, err, "handleSdkCall should fail block proposer is not accessible in signed txs")

				t.Log("getBlockCommittee")
				res, err = h.handleSdkCall(ctx, executionContextId, sdk.SDK_OPERATION_NAME_ENV, "getBlockCommittee")
				require.NoError(t, err, "handleSdkCall should not fail")
				committee := res[0].BytesArrayValueCopiedToNative()
				require.Len(t, committee, 4, "should be 4 elements")
				require.EqualValues(t, testKeys.NodeAddressesForTests()[3], committee[3], "handleSdkCall result should be equal")

				t.Log("getNextBlockCommittee")
				res, err = h.handleSdkCall(ctx, executionContextId, sdk.SDK_OPERATION_NAME_ENV, "getNextBlockCommittee")
				require.Error(t, err, "handleSdkCall should fail next committee is not accessible in signed txs")

				return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
			})

			h.processTransactionSetWithBlockInfo(ctx, currentBlockHeight, currentBlockTimestamp, currentBlockProposer, []*contractAndMethod{
				{"Contract1", "method1"},
			})

			h.verifySystemContractCalled(t)
			h.verifyNativeContractMethodCalled(t)
		})
	})
}

func TestSdkEnv_GetBlockDetails_InUnsignedTransaction(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {

			h := newHarness(parent.Logger)
			h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

			const currentBlockHeight = primitives.BlockHeight(12)
			const currentBlockTimestamp = primitives.TimestampNano(0x777)
			currentBlockProposer := hash.Make32BytesWithFirstByte(5)

			h.expectNativeContractMethodCalled("_Triggers", "trigger", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
				t.Log("getBlockHeight")
				res, err := h.handleSdkCall(ctx, executionContextId, sdk.SDK_OPERATION_NAME_ENV, "getBlockHeight")
				require.NoError(t, err, "handleSdkCall should not fail")
				require.Equal(t, uint64(currentBlockHeight), res[0].Uint64Value(), "handleSdkCall result should be equal")

				t.Log("getBlockTimestamp")
				res, err = h.handleSdkCall(ctx, executionContextId, sdk.SDK_OPERATION_NAME_ENV, "getBlockTimestamp")
				require.NoError(t, err, "handleSdkCall should not fail")
				require.Equal(t, uint64(currentBlockTimestamp), res[0].Uint64Value(), "handleSdkCall result should be equal")

				t.Log("getBlockProposerAddress")
				res, err = h.handleSdkCall(ctx, executionContextId, sdk.SDK_OPERATION_NAME_ENV, "getBlockProposerAddress")
				require.NoError(t, err, "handleSdkCall should not fail")
				require.EqualValues(t, currentBlockProposer, res[0].BytesValue(), "handleSdkCall result should be equal")

				t.Log("getBlockCommittee")
				res, err = h.handleSdkCall(ctx, executionContextId, sdk.SDK_OPERATION_NAME_ENV, "getBlockCommittee")
				require.NoError(t, err, "handleSdkCall should not fail")
				committee := res[0].BytesArrayValueCopiedToNative()
				require.Len(t, committee, 4, "should be 4 elements")
				require.EqualValues(t, testKeys.NodeAddressesForTests()[3], committee[3], "handleSdkCall result should be equal")

				t.Log("getNextBlockCommittee")
				res, err = h.handleSdkCall(ctx, executionContextId, sdk.SDK_OPERATION_NAME_ENV, "getNextBlockCommittee")
				nextCommittee := res[0].BytesArrayValueCopiedToNative()
				require.NoError(t, err, "handleSdkCall should not fail")
				require.Len(t, committee, 4, "should be 4 elements")
				require.EqualValues(t, testKeys.NodeAddressesForTests()[4], nextCommittee[3], "handleSdkCall result should be equal")

				return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
			})

			h.processTriggerTransaction(ctx, currentBlockHeight, currentBlockTimestamp, currentBlockProposer)
		})
	})
}

func TestSdkEnv_GetBlockDetails_InCallMethod(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {

			h := newHarness(parent.Logger)
			h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

			const lastCommittedBlockHeight = primitives.BlockHeight(12)
			const lastCommittedBlockTimestamp = primitives.TimestampNano(0x777)
			currentBlockProposer := hash.Make32BytesWithFirstByte(5)

			h.expectStateStorageLastCommittedBlockInfoRequested(lastCommittedBlockHeight, lastCommittedBlockTimestamp, currentBlockProposer)
			h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
				t.Log("getBlockHeight")
				res, err := h.handleSdkCall(ctx, executionContextId, sdk.SDK_OPERATION_NAME_ENV, "getBlockHeight")
				require.NoError(t, err, "handleSdkCall should not fail")
				require.Equal(t, uint64(lastCommittedBlockHeight), res[0].Uint64Value(), "handleSdkCall result should be equal")

				t.Log("getBlockTimestamp")
				res, err = h.handleSdkCall(ctx, executionContextId, sdk.SDK_OPERATION_NAME_ENV, "getBlockTimestamp")
				require.NoError(t, err, "handleSdkCall should not fail")
				require.Equal(t, uint64(lastCommittedBlockTimestamp), res[0].Uint64Value(), "handleSdkCall result should be equal")

				t.Log("getBlockProposerAddress")
				res, err = h.handleSdkCall(ctx, executionContextId, sdk.SDK_OPERATION_NAME_ENV, "getBlockProposerAddress")
				require.NoError(t, err, "handleSdkCall should not fail")
				require.EqualValues(t, currentBlockProposer, res[0].BytesValue(), "handleSdkCall result should be equal")

				return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
			})

			h.processQuery(ctx, "Contract1", "method1")

			h.verifySystemContractCalled(t)
			h.verifyStateStorageBlockHeightRequested(t)
			h.verifyNativeContractMethodCalled(t)
		})
	})
}
