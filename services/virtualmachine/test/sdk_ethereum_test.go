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
			res, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_ETHEREUM, "callMethod", "EthContractAddress", "EthJsonAbi", "EthMethodName", []byte{0x01, 0x02, 0x03})
			require.NoError(t, err, "handleSdkCall should not fail")
			require.Equal(t, []byte{0x04, 0x05, 0x06}, res[0].BytesValue(), "handleSdkCall result should be equal")
			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})
		h.expectEthereumConnectorMethodCalled("EthContractAddress", "EthMethodName", nil, []byte{0x04, 0x05, 0x06})

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
			res, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_ETHEREUM, "getTransactionLog", "EthContractAddress", "EthJsonAbi", []byte{0x01, 0x02, 0x03}, "EthEventName")
			require.NoError(t, err, "handleSdkCall should not fail")
			require.Equal(t, []byte{0x04, 0x05, 0x06}, res[0].BytesValue(), "handleSdkCall result should be equal")
			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})
		h.expectEthereumConnectorGetTransactionLogs("EthContractAddress", "EthEventName", nil, []byte{0x04, 0x05, 0x06})

		h.processTransactionSet(ctx, []*contractAndMethod{
			{"Contract1", "method1"},
		})

		h.verifySystemContractCalled(t)
		h.verifyNativeContractMethodCalled(t)
		h.verifyEthereumConnectorMethodCalled(t)
	})
}
