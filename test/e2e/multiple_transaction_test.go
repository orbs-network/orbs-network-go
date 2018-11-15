package e2e

import (
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNetworkCommitsMultipleTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	lt := time.Now()
	printTestTime(t, "started", &lt)

	h := newHarness()
	defer h.gracefulShutdown()
	printTestTime(t, "new harness", &lt) // slow do to warm up compilation

	// send 3 transactions with total of 70
	amounts := []uint64{15, 22, 33}
	for _, amount := range amounts {
		signerKeyPair := keys.Ed25519KeyPairForTests(5)
		targetAddress := builders.AddressForEd25519SignerForTests(6)
		transfer := builders.TransferTransaction().WithEd25519Signer(signerKeyPair).WithAmountAndTargetAddress(amount, targetAddress).Builder()

		printTestTime(t, "send transaction - start", &lt)
		response, err := h.sendTransaction(transfer)
		printTestTime(t, "send transaction - end", &lt)

		require.NoError(t, err, "transaction for amount %d should not return error", amount)
		test.RequireSuccess(t, response, "transaction for amount %d should be successfully committed and executed", amount)
	}

	// check balance
	ok := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
		signerKeyPair := keys.Ed25519KeyPairForTests(6)
		targetAddress := builders.AddressForEd25519SignerForTests(6)
		getBalance := builders.GetBalanceTransaction().WithEd25519Signer(signerKeyPair).WithTargetAddress(targetAddress).Builder().Transaction

		printTestTime(t, "call method - start", &lt)
		response, err := h.callMethod(getBalance)
		printTestTime(t, "call method - end", &lt)

		if err == nil && response.CallMethodResult() == protocol.EXECUTION_RESULT_SUCCESS {
			outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(response)
			if outputArgsIterator.HasNext() {
				return outputArgsIterator.NextArguments().Uint64Value() == 70
			}
		}
		return false
	})

	require.True(t, ok, "getBalance should return total amount")
	printTestTime(t, "done", &lt)
}
