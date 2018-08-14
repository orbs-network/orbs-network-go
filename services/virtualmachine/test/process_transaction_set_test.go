package test

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProcessTransactionSetSuccess(t *testing.T) {
	h := newHarness()

	h.expectNativeProcessorCalled(func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		t.Log("Transaction 1: successful")

		return protocol.EXECUTION_RESULT_SUCCESS, nil
	}, func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		t.Log("Transaction 2: successful")

		return protocol.EXECUTION_RESULT_SUCCESS, nil
	})

	results, _ := h.processTransactionSet([]primitives.ContractName{"ExampleContract", "ExampleContract"})
	require.Equal(t, results, []protocol.ExecutionResult{
		protocol.EXECUTION_RESULT_SUCCESS,
		protocol.EXECUTION_RESULT_SUCCESS,
	}, "processTransactionSet returned receipts should match")

	h.verifyNativeProcessorCalled(t)
}

func TestProcessTransactionSetWithErrors(t *testing.T) {
	h := newHarness()

	h.expectNativeProcessorCalled(func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		t.Log("Transaction 1: failed (contract error)")

		return protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT, errors.New("contract error")
	}, func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		t.Log("Transaction 2: failed (unexpected error)")

		return protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, errors.New("unexpected error")
	})

	results, _ := h.processTransactionSet([]primitives.ContractName{"ExampleContract", "ExampleContract"})
	require.Equal(t, results, []protocol.ExecutionResult{
		protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT,
		protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
	}, "processTransactionSet returned receipts should match")

	h.verifyNativeProcessorCalled(t)
}
