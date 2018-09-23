package e2e

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
)

func TestDeploymentOfNativeContract(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	h := newHarness()
	defer h.gracefulShutdown()

	counterStart := uint64(100 * rand.Intn(1000))

	// transaction to deploy the contract
	deploy := builders.Transaction().
		WithMethod("_Deployments", "deployService").
		WithArgs(
			fmt.Sprintf("CounterFrom%d", counterStart),
			uint32(protocol.PROCESSOR_TYPE_NATIVE),
			[]byte(contracts.NativeSourceCodeForCounter(counterStart)),
		).Builder()
	response, err := h.sendTransaction(t, deploy)
	require.NoError(t, err, "deploy transaction should not return error")
	require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, response.TransactionStatus(), "deploy transaction should be successfully committed")
	require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, response.TransactionReceipt().ExecutionResult(), "deploy transaction should execute successfully")

	// check counter
	ok := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
		getCounter := &protocol.TransactionBuilder{
			ContractName: primitives.ContractName(fmt.Sprintf("CounterFrom%d", counterStart)),
			MethodName:   "get",
		}
		response, err := h.callMethod(t, getCounter)
		if err == nil && response.CallMethodResult() == protocol.EXECUTION_RESULT_RESERVED { // TODO: this is a bug, change to EXECUTION_RESULT_SUCCESS
			outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(response)
			if outputArgsIterator.HasNext() {
				return outputArgsIterator.NextArguments().Uint64Value() == counterStart
			}
		}
		return false
	})
	require.True(t, ok, "get counter should return counter start")

	// transaction to add to the counter
	amount := uint64(17)
	add := builders.Transaction().
		WithMethod(primitives.ContractName(fmt.Sprintf("CounterFrom%d", counterStart)), "add").
		WithArgs(amount).
		Builder()
	response, err = h.sendTransaction(t, add)
	require.NoError(t, err, "add transaction should not return error")
	require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, response.TransactionStatus(), "add transaction should be successfully committed")
	require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, response.TransactionReceipt().ExecutionResult(), "add transaction should execute successfully")

	// check counter
	ok = test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
		getCounter := &protocol.TransactionBuilder{
			ContractName: primitives.ContractName(fmt.Sprintf("CounterFrom%d", counterStart)),
			MethodName:   "get",
		}
		response, err := h.callMethod(t, getCounter)
		if err == nil && response.CallMethodResult() == protocol.EXECUTION_RESULT_RESERVED { // TODO: this is a bug, change to EXECUTION_RESULT_SUCCESS
			outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(response)
			if outputArgsIterator.HasNext() {
				return outputArgsIterator.NextArguments().Uint64Value() == counterStart+amount
			}
		}
		return false
	})
	require.True(t, ok, "get counter should return counter start plus added value")
}
