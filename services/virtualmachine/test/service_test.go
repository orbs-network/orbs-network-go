package test

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInit(t *testing.T) {
	h := newHarness()
	h.verifyHandlerRegistrations(t)
}

func TestSdkUnknownContract(t *testing.T) {
	h := newHarness()

	h.expectStateStorageBlockHeightRequested(12)
	h.expectNativeProcessorCalled(func(contextId primitives.ExecutionContextId) protocol.ExecutionResult {
		_, err := h.handleSdkCall(contextId, "UnknownContract", "read")
		require.Error(t, err, "handleSdkCall should fail")
		return protocol.EXECUTION_RESULT_SUCCESS
	})

	h.runLocalMethod()

	h.verifyStateStorageBlockHeightRequested(t)
	h.verifyNativeProcessorCalled(t)
}
