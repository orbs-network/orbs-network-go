package e2e

import (
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	orbsClient "github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSendTransactionToTwoSeparateChains(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	appChain := NewAppHarness()
	mgmtChain := NewMgmtHarness()

	amount1 := uint64(13)
	amount2 := uint64(17)

	appChain.WaitUntilTransactionPoolIsReady(t)
	mgmtChain.WaitUntilTransactionPoolIsReady(t)

	recipient, _ := orbsClient.CreateAccount()

	response1, _, err1 := appChain.SendTransaction(OwnerOfAllSupply.PublicKey(), OwnerOfAllSupply.PrivateKey(), "BenchmarkToken", "transfer", amount1, recipient.AddressAsBytes())
	response2, _, err2 := mgmtChain.SendTransaction(OwnerOfAllSupply.PublicKey(), OwnerOfAllSupply.PrivateKey(), "BenchmarkToken", "transfer", amount2, recipient.AddressAsBytes())

	require.NoError(t, err1, "expected tx1 to succeed")
	requireSuccessful(t, response1)

	require.NoError(t, err2, "expected tx2 to succeed")
	requireSuccessful(t, response2)

	// check balance
	eventuallyBalance(t, appChain, recipient, amount1)
	eventuallyBalance(t, mgmtChain, recipient, amount2)
}

func eventuallyBalance(t *testing.T, chainHarness *Harness, address *orbsClient.OrbsAccount, expectedBalance uint64, msgAndArgs ...interface{}) {
	var lastObservedBalance uint64
	ok := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
		response, err := chainHarness.RunQuery(address.PublicKey, "BenchmarkToken", "getBalance", address.AddressAsBytes())

		if err != nil {
			return false
		}

		if response.ExecutionResult != codec.EXECUTION_RESULT_SUCCESS {
			return false
		}
		lastObservedBalance = response.OutputArguments[0].(uint64)
		return lastObservedBalance == expectedBalance
	})
	t.Log("found balance of", lastObservedBalance, "for address", address.Address)
	require.True(t, ok, msgAndArgs...)
}
