package acceptance

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/test/harness"
	. "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
	"time"
)

func TestCreateGazillionTransactionsWhileTransportIsDuplicatingRandomMessages(t *testing.T) {
	harness.Network(t).
		WithLogFilters(log.IgnoreMessagesMatching("leader failed to validate vote"), log.IgnoreErrorsMatching("transaction rejected: TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING")).
		WithNumNodes(3).Start(func(network harness.InProcessNetwork) {
		network.GossipTransport().Duplicate(AnyNthMessage(7))

		sendTransfersAndAssertTotalBalance(network, t, 100)
	})
}

func TestCreateGazillionTransactionsWhileTransportIsDroppingRandomMessages(t *testing.T) {
	harness.Network(t).
		WithLogFilters(log.IgnoreMessagesMatching("leader failed to validate vote"), log.IgnoreErrorsMatching("transport failed to send")).
		WithNumNodes(3).Start(func(network harness.InProcessNetwork) {
		network.GossipTransport().Fail(HasHeader(ABenchmarkConsensusMessage).And(AnyNthMessage(7)))

		sendTransfersAndAssertTotalBalance(network, t, 100)
	})
}

func TestCreateGazillionTransactionsWhileTransportIsDelayingRandomMessages(t *testing.T) {
	harness.Network(t).
		WithLogFilters(log.IgnoreMessagesMatching("leader failed to validate vote")).
		WithNumNodes(3).Start(func(network harness.InProcessNetwork) {

		network.GossipTransport().Delay(func() time.Duration {
			return (time.Duration(rand.Intn(1000)) + 1000) * time.Microsecond // delay each message between 1000 and 2000 millis
		}, AnyNthMessage(2))

		sendTransfersAndAssertTotalBalance(network, t, 100)
	})
}

func TestCreateGazillionTransactionsWhileTransportIsCorruptingRandomMessages(t *testing.T) {
	t.Skip("this test causes the system to hang, seems like consensus algo stops")
	harness.Network(t).WithNumNodes(3).Start(func(network harness.InProcessNetwork) {
		network.GossipTransport().Corrupt(Not(HasHeader(ATransactionRelayMessage)).And(AnyNthMessage(7)))

		sendTransfersAndAssertTotalBalance(network, t, 100)
	})
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

func sendTransfersAndAssertTotalBalance(network harness.InProcessNetwork, t *testing.T, numTransactions int) {
	fromAddress := 5
	toAddress := 6

	var expectedSum uint64 = 0
	var txHashes []primitives.Sha256
	for i := 0; i < numTransactions; i++ {
		amount := uint64(rand.Int63n(100))
		expectedSum += amount

		txHash := network.SendTransferInBackground(rand.Intn(network.Size()), amount, fromAddress, toAddress)
		txHashes = append(txHashes, txHash)
	}
	for _, txHash := range txHashes {
		for i := 0; i < network.Size(); i++ {
			network.WaitForTransactionInState(i, txHash)
		}
	}

	for i := 0; i < network.Size(); i++ {
		actualSum := <-network.CallGetBalance(i, toAddress)

		require.EqualValuesf(t, expectedSum, actualSum, "balance did not equal expected balance in node %d", i)
	}
}
