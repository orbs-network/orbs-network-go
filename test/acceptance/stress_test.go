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

func TestCreateGazillionTransactionsWhileTransportIsDroppingRandomMessages(t *testing.T) {
	t.Parallel()
	harness.Network(t).WithNumNodes(4).Start(func(network harness.AcceptanceTestNetwork) {
		network.GossipTransport().Fail(HasHeader(ABenchmarkConsensusMessage).And(AnyNthMessage(7)))

		send100Transactions(network, t)
	})
}

func TestCreateGazillionTransactionsWhileTransportIsDuplicatingRandomMessages(t *testing.T) {
	t.Parallel()
	harness.Network(t).WithNumNodes(4).Start(func(network harness.AcceptanceTestNetwork) {
		network.GossipTransport().Duplicate(AnyNthMessage(7))

		send100Transactions(network, t)
	})
}

func TestCreateGazillionTransactionsWhileTransportIsCorruptingRandomMessages(t *testing.T) {
	t.Parallel()
	harness.Network(t).WithNumNodes(3).Start(func(network harness.AcceptanceTestNetwork) {
		network.GossipTransport().Corrupt(Not(HasHeader(ATransactionRelayMessage)).And(AnyNthMessage(7)))

		send100Transactions(network, t)
	})
}

func send100Transactions(network harness.AcceptanceTestNetwork, t *testing.T) {
	var expectedSum uint64 = 0
	var txHashes []primitives.Sha256
	for i := 0; i < 100; i++ {
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

func AnyNthMessage(max int) MessagePredicate {
	count := 0
	return func(data *adapter.TransportData) bool {
		count++
		return count % max == 0
	}
}