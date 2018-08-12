package test

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRunLocalMethodSuccess(t *testing.T) {
	h := newHarness()

	h.expectStateStorageBlockHeightRequested(12)
	h.expectNativeProcessorCalled(func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		return protocol.EXECUTION_RESULT_SUCCESS, nil
	})

	result, refHeight, err := h.runLocalMethod("ExampleContract")
	require.NoError(t, err, "run local method should not fail")
	require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, result, "run local method should return successful result")
	require.EqualValues(t, 12, refHeight)

	h.verifyStateStorageBlockHeightRequested(t)
	h.verifyNativeProcessorCalled(t)
}

func TestRunLocalMethodContractError(t *testing.T) {
	h := newHarness()

	h.expectStateStorageBlockHeightRequested(12)
	h.expectNativeProcessorCalled(func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		return protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT, errors.New("contract error")
	})

	result, refHeight, err := h.runLocalMethod("ExampleContract")
	require.Error(t, err, "run local method should fail")
	require.Equal(t, protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT, result, "run local method should return contract error")
	require.EqualValues(t, 12, refHeight)

	h.verifyStateStorageBlockHeightRequested(t)
	h.verifyNativeProcessorCalled(t)
}

func TestRunLocalMethodUnexpectedError(t *testing.T) {
	h := newHarness()

	h.expectStateStorageBlockHeightRequested(12)
	h.expectNativeProcessorCalled(func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		return protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, errors.New("unexpected error")
	})

	result, refHeight, err := h.runLocalMethod("ExampleContract")
	require.Error(t, err, "run local method should fail")
	require.Equal(t, protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, result, "run local method should return unexpected error")
	require.EqualValues(t, 12, refHeight)

	h.verifyStateStorageBlockHeightRequested(t)
	h.verifyNativeProcessorCalled(t)
}
