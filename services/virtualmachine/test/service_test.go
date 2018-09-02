package test

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInit(t *testing.T) {
	h := newHarness()
	h.verifyHandlerRegistrations(t)
}

func TestSdkUnknownOperation(t *testing.T) {
	h := newHarness()
	h.expectSystemContractCalled(deployments.CONTRACT.Name, deployments.METHOD_GET_INFO.Name, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

	h.expectStateStorageBlockHeightRequested(12)
	h.expectNativeContractMethodCalled("Contract1", "method1", func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, *protocol.MethodArgumentArray, error) {
		_, err := h.handleSdkCall(contextId, "Sdk.UnknownOperation", "read", protocol.PERMISSION_SCOPE_SERVICE)
		require.Error(t, err, "handleSdkCall should fail")
		return protocol.EXECUTION_RESULT_SUCCESS, builders.MethodArgumentsArray(), nil
	})

	h.runLocalMethod("Contract1", "method1")

	h.verifySystemContractCalled(t)
	h.verifyStateStorageBlockHeightRequested(t)
	h.verifyNativeContractMethodCalled(t)
}
