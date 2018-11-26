package e2e

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestDeploymentOfNativeContract(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	runMultipleTimes(t, func(t *testing.T) {

		h := newHarness()
		lt := time.Now()
		printTestTime(t, "started", &lt)

		counterStart := uint64(time.Now().UnixNano())

		printTestTime(t, "send deploy - start", &lt)
		response, err := h.deployNativeContract(fmt.Sprintf("CounterFrom%d", counterStart), []byte(contracts.NativeSourceCodeForCounter(counterStart)))
		printTestTime(t, "send deploy - end", &lt)

		require.NoError(t, err, "deploy transaction should not return error")
		test.RequireSuccess(t, response, "deploy transaction should be successfully committed and executed")

		// check counter
		ok := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
			getCounter := builders.NonSignedTransaction().
				WithMethod(primitives.ContractName(fmt.Sprintf("CounterFrom%d", counterStart)), "get")

			printTestTime(t, "call method - start", &lt)
			response, err2 := h.callMethod(getCounter.Builder())
			printTestTime(t, "call method - end", &lt)

			if err2 == nil && response.CallMethodResult() == protocol.EXECUTION_RESULT_SUCCESS {
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

		printTestTime(t, "send transaction - start", &lt)
		response, err = h.sendTransaction(add)
		printTestTime(t, "send transaction - end", &lt)

		require.NoError(t, err, "add transaction should not return error")
		test.RequireSuccess(t, response, "add transaction should be successfully committed and executed")

		// check counter
		ok = test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
			getCounter := builders.NonSignedTransaction().
				WithMethod(primitives.ContractName(fmt.Sprintf("CounterFrom%d", counterStart)), "get")

			response, err := h.callMethod(getCounter.Builder())
			if err == nil && response.CallMethodResult() == protocol.EXECUTION_RESULT_SUCCESS {
				outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(response)
				if outputArgsIterator.HasNext() {
					return outputArgsIterator.NextArguments().Uint64Value() == counterStart+amount
				}
			}
			return false
		})

		require.True(t, ok, "get counter should return counter start plus added value")
		printTestTime(t, "done", &lt)

	})
}
