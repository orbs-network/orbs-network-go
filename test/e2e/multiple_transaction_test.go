package e2e

import (
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNetworkCommitsMultipleTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	h := newHarness()
	defer h.gracefulShutdown()

	// send 3 transactions with total of 70
	amounts := []uint64{15, 22, 33}
	for _, amount := range amounts {
		transfer := builders.TransferTransaction().WithAmount(amount).Builder()
		response, err := h.sendTransaction(t, transfer)
		require.NoError(t, err, "transaction for amount %d should not return error", amount)
		require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, response.TransactionStatus(), "transaction for amount %d should be successfully committed", amount)
		require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, response.TransactionReceipt().ExecutionResult(), "transaction for amount %d should execute successfully", amount)
	}

	// check balance
	getBalance := &protocol.TransactionBuilder{
		ContractName: "BenchmarkToken",
		MethodName:   "getBalance",
	}

	ok := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
		response, err := h.callMethod(t, getBalance)
		if err == nil && response.CallResult() == protocol.EXECUTION_RESULT_RESERVED { // TODO: this is a bug, change to EXECUTION_RESULT_SUCCESS
			outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(response)
			if outputArgsIterator.HasNext() {
				return outputArgsIterator.NextArguments().Uint64Value() == 70
			}
		}
		return false
	})
	require.True(t, ok, "getBalance should return total amount")
}
