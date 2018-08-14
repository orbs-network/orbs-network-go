package test

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSdkReadWithoutContext(t *testing.T) {
	h := newHarness()

	_, err := h.handleSdkCall(999, native.SDK_STATE_CONTRACT_NAME, "read", []byte{0x01})
	require.Error(t, err, "handleSdkCall should fail")
}

func TestSdkReadStateWithLocalMethodReadOnlyAccess(t *testing.T) {
	h := newHarness()

	h.expectStateStorageBlockHeightRequested(12)
	h.expectNativeProcessorCalled(func(contextId primitives.ExecutionContextId) protocol.ExecutionResult {

		t.Log("First read should reach state storage")

		res, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "read", []byte{0x01})
		require.NoError(t, err, "handleSdkCall should not fail")
		require.Equal(t, []byte{0x02}, res[0].BytesValue(), "handleSdkCall result should be equal")

		t.Log("Second read should be cached")

		res, err = h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "read", []byte{0x01})
		require.NoError(t, err, "handleSdkCall should not fail")
		require.Equal(t, []byte{0x02}, res[0].BytesValue(), "handleSdkCall result should be equal")

		return protocol.EXECUTION_RESULT_SUCCESS
	})
	h.expectStateStorageRead(12, []byte{0x01}, []byte{0x02})

	h.runLocalMethod()

	h.verifyStateStorageBlockHeightRequested(t)
	h.verifyNativeProcessorCalled(t)
	h.verifyStateStorageRead(t)
}

func TestSdkWriteStateWithLocalMethodReadOnlyAccess(t *testing.T) {
	h := newHarness()

	h.expectStateStorageBlockHeightRequested(12)
	h.expectNativeProcessorCalled(func(contextId primitives.ExecutionContextId) protocol.ExecutionResult {

		t.Log("Attempt to write without proper access")

		_, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "write", []byte{0x01}, []byte{0x02})
		require.Error(t, err, "handleSdkCall should fail")

		return protocol.EXECUTION_RESULT_ERROR_UNEXPECTED
	})

	h.runLocalMethod()

	h.verifyStateStorageBlockHeightRequested(t)
	h.verifyNativeProcessorCalled(t)
}

func TestSdkWriteStateWithTransactionSetReadWriteAccess(t *testing.T) {
	h := newHarness()

	h.expectNativeProcessorCalled(func(contextId primitives.ExecutionContextId) protocol.ExecutionResult {

		t.Log("Transaction 1: first write should change in transient state")

		_, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "write", []byte{0x01}, []byte{0x02})
		require.NoError(t, err, "handleSdkCall should succeed")

		t.Log("Transaction 1: second write should replace in transient state")

		_, err = h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "write", []byte{0x01}, []byte{0x03, 0x04})
		require.NoError(t, err, "handleSdkCall should succeed")

		return protocol.EXECUTION_RESULT_SUCCESS
	}, func(contextId primitives.ExecutionContextId) protocol.ExecutionResult {

		t.Log("Transaction 2: first write should replace in transient state")

		_, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "write", []byte{0x01}, []byte{0x05, 0x06})
		require.NoError(t, err, "handleSdkCall should succeed")

		t.Log("Transaction 2: read should return last successful write")

		res, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "read", []byte{0x01})
		require.NoError(t, err, "handleSdkCall should not fail")
		require.Equal(t, []byte{0x05, 0x06}, res[0].BytesValue(), "handleSdkCall result should be equal")

		return protocol.EXECUTION_RESULT_SUCCESS
	})
	h.expectStateStorageNotRead()

	sd := h.processTransactionSet(2)
	require.ElementsMatch(t, sd, []*keyValuePair{
		{[]byte{0x01}, []byte{0x05, 0x06}},
	}, "processTransactionSet returned contract state diffs should match")

	h.verifyNativeProcessorCalled(t)
	h.verifyStateStorageRead(t)
}

func TestSdkWriteStateIgnoredWithTransactionSetHavingFailedTransactions(t *testing.T) {
	h := newHarness()

	h.expectNativeProcessorCalled(func(contextId primitives.ExecutionContextId) protocol.ExecutionResult {

		t.Log("Transaction 1 (successful): first write should change in transient state")

		_, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "write", []byte{0x01}, []byte{0x02})
		require.NoError(t, err, "handleSdkCall should succeed")

		return protocol.EXECUTION_RESULT_SUCCESS
	}, func(contextId primitives.ExecutionContextId) protocol.ExecutionResult {

		t.Log("Transaction 2 (failed): write should be ignored")

		_, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "write", []byte{0x01}, []byte{0x03, 0x04})
		require.NoError(t, err, "handleSdkCall should succeed")

		t.Log("Transaction 2 (failed): read the ignored write should return it")

		res, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "read", []byte{0x01})
		require.NoError(t, err, "handleSdkCall should not fail")
		require.Equal(t, []byte{0x03, 0x04}, res[0].BytesValue(), "handleSdkCall result should be equal")

		return protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT
	}, func(contextId primitives.ExecutionContextId) protocol.ExecutionResult {

		t.Log("Transaction 3 (successful): read should return last successful write")

		res, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "read", []byte{0x01})
		require.NoError(t, err, "handleSdkCall should not fail")
		require.Equal(t, []byte{0x02}, res[0].BytesValue(), "handleSdkCall result should be equal")

		return protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT
	})
	h.expectStateStorageNotRead()

	sd := h.processTransactionSet(3)
	require.ElementsMatch(t, sd, []*keyValuePair{
		{[]byte{0x01}, []byte{0x02}},
	}, "processTransactionSet returned contract state diffs should be empty")

	h.verifyNativeProcessorCalled(t)
	h.verifyStateStorageRead(t)
}
