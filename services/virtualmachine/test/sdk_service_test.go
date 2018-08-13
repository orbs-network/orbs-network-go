package test

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSdkServiceIsNative(t *testing.T) {
	h := newHarness()

	h.expectStateStorageBlockHeightRequested(12)
	h.expectContractMethodCalled("Contract1", "method1", func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		t.Log("First isNative on unknown contract")

		_, err := h.handleSdkCall(contextId, native.SDK_SERVICE_CONTRACT_NAME, "isNative", "UnknownContract")
		require.Error(t, err, "handleSdkCall should fail")

		t.Log("Second isNative on known contract")

		_, err = h.handleSdkCall(contextId, native.SDK_SERVICE_CONTRACT_NAME, "isNative", "NativeContract")
		require.NoError(t, err, "handleSdkCall should not fail")

		return protocol.EXECUTION_RESULT_SUCCESS, nil
	})
	h.expectContractInfoRequested("UnknownContract", errors.New("unknown contract"))
	h.expectContractInfoRequested("NativeContract", nil)

	h.runLocalMethod("Contract1", "method1")

	h.verifyStateStorageBlockHeightRequested(t)
	h.verifyContractMethodCalled(t)
	h.verifyContractInfoRequested(t)
}
