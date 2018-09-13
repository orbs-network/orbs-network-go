package acceptance

import (
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/test/harness"
	. "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
)

func TestCreateGazillionTransactionsWhileTransportIsDuplicatingRandomMessages(t *testing.T) {
	harness.Network(t).WithNumNodes(4).Start(func(network harness.AcceptanceTestNetwork) {
		network.GossipTransport().Duplicate(AnyNthMessage(7))

		sendTransactions(network, t, 100)
	})
}

func TestCreateGazillionTransactionsWhileTransportIsDroppingRandomMessages(t *testing.T) {
	harness.Network(t).WithNumNodes(4).Start(func(network harness.AcceptanceTestNetwork) {
		network.GossipTransport().Fail(HasHeader(ABenchmarkConsensusMessage).And(AnyNthMessage(7)))

		sendTransactions(network, t, 100)
	})
}

func TestCreateGazillionTransactionsWhileTransportIsDelayingRandomMessages(t *testing.T) {
	harness.Network(t).WithNumNodes(4).Start(func(network harness.AcceptanceTestNetwork) {
		network.GossipTransport().Delay(AnyNthMessage(1))

		sendTransactions(network, t, 100)
	})
}

func TestCreateGazillionTransactionsWhileTransportIsCorruptingRandomMessages(t *testing.T) {
	t.Skip("this test causes the system to hang, seems like consensus algo stops")
	harness.Network(t).WithNumNodes(3).Start(func(network harness.AcceptanceTestNetwork) {
		network.GossipTransport().Corrupt(Not(HasHeader(ATransactionRelayMessage)).And(AnyNthMessage(7)))

		sendTransactions(network, t, 100)
	})
}

func sendTransactions(network harness.AcceptanceTestNetwork, t *testing.T, numTransactions int) {
	var expectedSum uint64 = 0
	var txHashes []primitives.Sha256
	for i := 0; i < numTransactions; i++ {
		amount := uint64(rand.Int63n(100))
		expectedSum += amount

		txHash := network.SendTransferInBackground(rand.Intn(network.Size()), amount)
		txHashes = append(txHashes, txHash)
	}
	for _, txHash := range txHashes {
		for i := 0; i < network.Size(); i++ {
			network.WaitForTransactionInState(i, txHash)
		}
	}

	for i := 0; i < network.Size(); i++ {
		require.EqualValuesf(t, expectedSum, <-network.CallGetBalance(i), "balance did not equal expected balance in node %d", i)
	}
}

func AnyNthMessage(n int) MessagePredicate {
	if n < 1 {
		panic("illegal argument")
	}

	if n == 1 {
		return func(data *adapter.TransportData) bool {
			return true
		}
	}

	count := 0
	return func(data *adapter.TransportData) bool {
		count++
		m := count % n
		return m == 0
	}
}