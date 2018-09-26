package e2e

import (
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
)

func sendAmount(t *testing.T, h *harness, amount uint64) {
	signerKeyPair := keys.Ed25519KeyPairForTests(5)
	targetAddress := builders.AddressForEd25519SignerForTests(6)
	transfer := builders.TransferTransaction().WithEd25519Signer(signerKeyPair).WithAmountAndTargetAddress(amount, targetAddress).Builder()
	response, err := h.sendTransaction(t, transfer)
	require.NoError(t, err, "transaction for amount %d should not return error", amount)
	require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, response.TransactionStatus(), "transaction for amount %d should be successfully committed", amount)
	require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, response.TransactionReceipt().ExecutionResult(), "transaction for amount %d should execute successfully", amount)

}

func TestNetworkCommitsMultipleTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	h := newHarness()
	defer h.gracefulShutdown()

	// send 3 transactions with total of 70
	amounts := []uint64{15, 22, 33}
	for _, amount := range amounts {
		sendAmount(t, h, amount)
	}

	// check balance
	ok := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
		signerKeyPair := keys.Ed25519KeyPairForTests(6)
		targetAddress := builders.AddressForEd25519SignerForTests(6)
		getBalance := builders.GetBalanceTransaction().WithEd25519Signer(signerKeyPair).WithTargetAddress(targetAddress).Builder().Transaction
		response, err := h.callMethod(t, getBalance)
		if err == nil && response.CallMethodResult() == protocol.EXECUTION_RESULT_RESERVED { // TODO: this is a bug, change to EXECUTION_RESULT_SUCCESS
			outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(response)
			if outputArgsIterator.HasNext() {
				return outputArgsIterator.NextArguments().Uint64Value() == 70
			}
		}
		return false
	})
	require.True(t, ok, "getBalance should return total amount")
}

func TestMultipleTransactionsProcessingTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	h := newHarness()
	defer h.gracefulShutdown()

	const numOfBrackets = 10
	const txPerBracket = 1

	amounts := [numOfBrackets][txPerBracket]uint64{}
	sum := uint64(0)

	for i := 0; i < numOfBrackets; i++ {
		for j := 0; j < txPerBracket; j++ {
			amount := uint64(rand.Intn(20))
			amounts[i][j] = amount
			sum += amount
		}
	}

	sendAmount(t, h, 0)

	for _, bracket := range amounts {
		for _, amount := range bracket {
			go func() {
				sendAmount(t, h, amount)
			}()
		}
	}

	// check balance
	ok := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
		signerKeyPair := keys.Ed25519KeyPairForTests(6)
		targetAddress := builders.AddressForEd25519SignerForTests(6)
		getBalance := builders.GetBalanceTransaction().WithEd25519Signer(signerKeyPair).WithTargetAddress(targetAddress).Builder().Transaction
		response, err := h.callMethod(t, getBalance)
		if err == nil && response.CallMethodResult() == protocol.EXECUTION_RESULT_RESERVED { // TODO: this is a bug, change to EXECUTION_RESULT_SUCCESS
			outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(response)
			if outputArgsIterator.HasNext() {
				val := outputArgsIterator.NextArguments().Uint64Value()
				return val == sum
			}
		}
		return false
	})
	require.True(t, ok, "getBalance should return total amount")
}
