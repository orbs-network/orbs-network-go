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

func TestSdkEthereum_CallMethod(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Ethereum callMethod")
			res, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_ETHEREUM, "callMethod", "EthContractAddress", "EthJsonAbi", uint64(1234), "EthMethodName", []byte{0x01, 0x02, 0x03})
			require.NoError(t, err, "handleSdkCall should not fail")
			require.Equal(t, []byte{0x04, 0x05, 0x06}, res[0].BytesValue(), "handleSdkCall result should be equal")
			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})
		h.expectEthereumConnectorMethodCalled("EthContractAddress", 1234, "EthMethodName", nil, []byte{0x04, 0x05, 0x06})

		h.processTransactionSet(ctx, []*contractAndMethod{
			{"Contract1", "method1"},
		})

		h.verifySystemContractCalled(t)
		h.verifyNativeContractMethodCalled(t)
		h.verifyEthereumConnectorMethodCalled(t)
	})
}

func TestSdkEthereum_GetTransactionLog(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Ethereum getTransactionLog")
			res, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_ETHEREUM, "getTransactionLog", "EthContractAddress", "EthJsonAbi", "EthTxHash", "EthEventName")
			require.NoError(t, err, "handleSdkCall should not fail")
			require.Equal(t, []byte{0x04, 0x05, 0x06}, res[0].BytesValue(), "handleSdkCall result should be equal")
			require.Equal(t, uint64(1234), res[1].Uint64Value(), "handleSdkCall block number result should be equal")
			require.Equal(t, uint32(56), res[2].Uint32Value(), "handleSdkCall txIndex result should be equal")
			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})
		h.expectEthereumConnectorGetTransactionLogs("EthContractAddress", "EthEventName", "EthTxHash", nil, []byte{0x04, 0x05, 0x06}, 1234, 56)

		h.processTransactionSet(ctx, []*contractAndMethod{
			{"Contract1", "method1"},
		})

		h.verifySystemContractCalled(t)
		h.verifyNativeContractMethodCalled(t)
		h.verifyEthereumConnectorMethodCalled(t)
	})
}

func TestSdkEthereum_GetBlockNumber(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Ethereum getBlockNumber")
			res, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_ETHEREUM, "getBlockNumber")
			require.NoError(t, err, "handleSdkCall should not fail")
			require.Equal(t, uint64(1234), res[0].Uint64Value(), "handleSdkCall block number result should be equal")
			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})
		h.expectEthereumConnectorGetBlockNumber(nil, 1234)

		h.processTransactionSet(ctx, []*contractAndMethod{
			{"Contract1", "method1"},
		})

		h.verifySystemContractCalled(t)
		h.verifyNativeContractMethodCalled(t)
		h.verifyEthereumConnectorMethodCalled(t)
	})
}
