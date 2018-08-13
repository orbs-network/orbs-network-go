package test

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSdkStateReadWithoutContext(t *testing.T) {
	h := newHarness()

	_, err := h.handleSdkCall(999, native.SDK_STATE_CONTRACT_NAME, "read", []byte{0x01})
	require.Error(t, err, "handleSdkCall should fail")
}

func TestSdkStateReadWithLocalMethodReadOnlyAccess(t *testing.T) {
	h := newHarness()

	h.expectStateStorageBlockHeightRequested(12)
	h.expectNativeContractMethodCalled("Contract1", "method1", func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		t.Log("First read should reach state storage")

		res, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "read", []byte{0x01})
		require.NoError(t, err, "handleSdkCall should not fail")
		require.Equal(t, []byte{0x02}, res[0].BytesValue(), "handleSdkCall result should be equal")

		t.Log("Second read should be from cache")

		res, err = h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "read", []byte{0x01})
		require.NoError(t, err, "handleSdkCall should not fail")
		require.Equal(t, []byte{0x02}, res[0].BytesValue(), "handleSdkCall result should be equal")

		return protocol.EXECUTION_RESULT_SUCCESS, nil
	})
	h.expectStateStorageRead(12, "Contract1", []byte{0x01}, []byte{0x02})

	h.runLocalMethod("Contract1", "method1")

	h.verifyStateStorageBlockHeightRequested(t)
	h.verifyNativeContractMethodCalled(t)
	h.verifyStateStorageRead(t)
}

func TestSdkStateWriteWithLocalMethodReadOnlyAccess(t *testing.T) {
	h := newHarness()

	h.expectStateStorageBlockHeightRequested(12)
	h.expectNativeContractMethodCalled("Contract1", "method1", func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		t.Log("Attempt to write without proper access")

		_, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "write", []byte{0x01}, []byte{0x02})
		require.Error(t, err, "handleSdkCall should fail")

		return protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, errors.New("unexpected error")
	})

	h.runLocalMethod("Contract1", "method1")

	h.verifyStateStorageBlockHeightRequested(t)
	h.verifyNativeContractMethodCalled(t)
}

func TestSdkStateReadWithTransactionSetReadWriteAccess(t *testing.T) {
	h := newHarness()

	h.expectNativeContractMethodCalled("Contract1", "method1", func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		t.Log("First read should reach state storage")

		res, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "read", []byte{0x01})
		require.NoError(t, err, "handleSdkCall should not fail")
		require.Equal(t, []byte{0x02}, res[0].BytesValue(), "handleSdkCall result should be equal")

		t.Log("Second read should be from cache")

		res, err = h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "read", []byte{0x01})
		require.NoError(t, err, "handleSdkCall should not fail")
		require.Equal(t, []byte{0x02}, res[0].BytesValue(), "handleSdkCall result should be equal")

		return protocol.EXECUTION_RESULT_SUCCESS, nil
	})
	h.expectStateStorageRead(11, "Contract1", []byte{0x01}, []byte{0x02})

	_, sd := h.processTransactionSet([]*contractAndMethod{
		{"Contract1", "method1"},
	})
	require.ElementsMatch(t, sd["Contract1"], []*keyValuePair{}, "processTransactionSet returned contract state diffs should be empty")

	h.verifyNativeContractMethodCalled(t)
	h.verifyStateStorageRead(t)
}

func TestSdkStateWriteWithTransactionSetReadWriteAccess(t *testing.T) {
	h := newHarness()

	h.expectNativeContractMethodCalled("Contract1", "method1", func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		t.Log("Transaction 1: first write should change in transient state")

		_, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "write", []byte{0x01}, []byte{0x02})
		require.NoError(t, err, "handleSdkCall should succeed")

		t.Log("Transaction 1: second write should replace in transient state")

		_, err = h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "write", []byte{0x01}, []byte{0x03, 0x04})
		require.NoError(t, err, "handleSdkCall should succeed")

		return protocol.EXECUTION_RESULT_SUCCESS, nil
	})
	h.expectNativeContractMethodCalled("Contract1", "method2", func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		t.Log("Transaction 2: first write should replace in transient state")

		_, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "write", []byte{0x01}, []byte{0x05, 0x06})
		require.NoError(t, err, "handleSdkCall should succeed")

		t.Log("Transaction 2: read should return last successful write")

		res, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "read", []byte{0x01})
		require.NoError(t, err, "handleSdkCall should not fail")
		require.Equal(t, []byte{0x05, 0x06}, res[0].BytesValue(), "handleSdkCall result should be equal")

		return protocol.EXECUTION_RESULT_SUCCESS, nil
	})
	h.expectStateStorageNotRead()

	_, sd := h.processTransactionSet([]*contractAndMethod{
		{"Contract1", "method1"},
		{"Contract1", "method2"},
	})
	require.ElementsMatch(t, sd["Contract1"], []*keyValuePair{
		{[]byte{0x01}, []byte{0x05, 0x06}},
	}, "processTransactionSet returned contract state diffs should match")

	h.verifyNativeContractMethodCalled(t)
	h.verifyStateStorageRead(t)
}

func TestSdkStateWriteOfDifferentContractsDoNotOverrideEachOther(t *testing.T) {
	h := newHarness()

	h.expectNativeContractMethodCalled("Contract1", "method1", func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		t.Log("Transaction 1: write to key in first contract")

		_, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "write", []byte{0x01}, []byte{0x02})
		require.NoError(t, err, "handleSdkCall should succeed")

		return protocol.EXECUTION_RESULT_SUCCESS, nil
	})
	h.expectNativeContractMethodCalled("Contract2", "method1", func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		t.Log("Transaction 2: write to same key in second contract")

		_, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "write", []byte{0x01}, []byte{0x03, 0x04})
		require.NoError(t, err, "handleSdkCall should succeed")

		return protocol.EXECUTION_RESULT_SUCCESS, nil
	})
	h.expectNativeContractMethodCalled("Contract1", "method2", func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		t.Log("Transaction 3: read from first contract")

		res, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "read", []byte{0x01})
		require.NoError(t, err, "handleSdkCall should not fail")
		require.Equal(t, []byte{0x02}, res[0].BytesValue(), "handleSdkCall result should be equal")

		return protocol.EXECUTION_RESULT_SUCCESS, nil
	})
	h.expectNativeContractMethodCalled("Contract2", "method2", func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		t.Log("Transaction 4: read from second contract")

		res, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "read", []byte{0x01})
		require.NoError(t, err, "handleSdkCall should not fail")
		require.Equal(t, []byte{0x03, 0x04}, res[0].BytesValue(), "handleSdkCall result should be equal")

		return protocol.EXECUTION_RESULT_SUCCESS, nil
	})
	h.expectStateStorageNotRead()

	_, sd := h.processTransactionSet([]*contractAndMethod{
		{"Contract1", "method1"},
		{"Contract2", "method1"},
		{"Contract1", "method2"},
		{"Contract2", "method2"},
	})
	require.ElementsMatch(t, sd["Contract1"], []*keyValuePair{
		{[]byte{0x01}, []byte{0x02}},
	}, "processTransactionSet returned contract state diffs should match")
	require.ElementsMatch(t, sd["Contract2"], []*keyValuePair{
		{[]byte{0x01}, []byte{0x03, 0x04}},
	}, "processTransactionSet returned contract state diffs should match")

	h.verifyNativeContractMethodCalled(t)
	h.verifyStateStorageRead(t)
}

func TestSdkStateWriteIgnoredWithTransactionSetHavingFailedTransactions(t *testing.T) {
	h := newHarness()

	h.expectNativeContractMethodCalled("Contract1", "method1", func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		t.Log("Transaction 1 (successful): first write should change in transient state")

		_, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "write", []byte{0x01}, []byte{0x02})
		require.NoError(t, err, "handleSdkCall should succeed")

		return protocol.EXECUTION_RESULT_SUCCESS, nil
	})
	h.expectNativeContractMethodCalled("Contract1", "method2", func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		t.Log("Transaction 2 (failed): write should be ignored")

		_, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "write", []byte{0x01}, []byte{0x03, 0x04})
		require.NoError(t, err, "handleSdkCall should succeed")

		t.Log("Transaction 2 (failed): read the ignored write should return it")

		res, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "read", []byte{0x01})
		require.NoError(t, err, "handleSdkCall should not fail")
		require.Equal(t, []byte{0x03, 0x04}, res[0].BytesValue(), "handleSdkCall result should be equal")

		return protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT, errors.New("contract error")
	})
	h.expectNativeContractMethodCalled("Contract1", "method3", func(contextId primitives.ExecutionContextId) (protocol.ExecutionResult, error) {
		t.Log("Transaction 3 (successful): read should return last successful write")

		res, err := h.handleSdkCall(contextId, native.SDK_STATE_CONTRACT_NAME, "read", []byte{0x01})
		require.NoError(t, err, "handleSdkCall should not fail")
		require.Equal(t, []byte{0x02}, res[0].BytesValue(), "handleSdkCall result should be equal")

		return protocol.EXECUTION_RESULT_SUCCESS, nil
	})
	h.expectStateStorageNotRead()

	_, sd := h.processTransactionSet([]*contractAndMethod{
		{"Contract1", "method1"},
		{"Contract1", "method2"},
		{"Contract1", "method3"},
	})
	require.ElementsMatch(t, sd["Contract1"], []*keyValuePair{
		{[]byte{0x01}, []byte{0x02}},
	}, "processTransactionSet returned contract state diffs should match")

	h.verifyNativeContractMethodCalled(t)
	h.verifyStateStorageRead(t)
}
