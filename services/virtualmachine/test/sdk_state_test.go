package test

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSdkReadWithoutContext(t *testing.T) {
	h := newHarness()

	_, err := h.handleSdkCall(999, native.SDK_STATE_CONTRACT_NAME, "read", []byte{0x01})
	require.Error(t, err, "handleSdkCall should fail")
}

func TestSdkReadStateWithoutTransientState(t *testing.T) {
	h := newHarness()

	h.expectStateStorageBlockHeightRequested(12)
	h.expectNativeProcessorCalled(func(contextId primitives.ExecutionContextId) {
		res, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "read", []byte{0x01})
		require.NoError(t, err, "handleSdkCall should not fail")
		require.Equal(t, []byte{0x02}, res[0].BytesValue(), "handleSdkCall result should be equal")
	})
	h.expectStateStorageRead(12, []byte{0x01}, []byte{0x02})

	h.runLocalMethod()

	h.verifyStateStorageBlockHeightRequested(t)
	h.verifyNativeProcessorCalled(t)
	h.verifyStateStorageRead(t)
}
