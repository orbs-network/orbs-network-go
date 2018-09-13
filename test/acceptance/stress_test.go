package acceptance

import (
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
)

func TestCreateSomeTransactions(t *testing.T) {
	harness.Network(t).Start(func(network harness.AcceptanceTestNetwork) {
		var expectedSum uint64 = 0
		for i := 0; i < 100; i++ {
			amount := uint64(rand.Int63n(100))
			expectedSum += amount

			result := <-network.SendTransfer(rand.Intn(network.Size()), amount)
			require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, result.TransactionStatus(), "transaction was not committed")
		}

		//time.Sleep(10 * time.Millisecond) - this sleep makes the tests pass consistently

		require.EqualValues(t, expectedSum, <-network.CallGetBalance(0), "balance did not equal expected balance in leader")
		require.EqualValues(t, expectedSum, <-network.CallGetBalance(1), "balance did not equal expected balance in validator")

	})
}

// TestCreateGazillionTransactionsWhileTransportIsDroppingRandomMessages