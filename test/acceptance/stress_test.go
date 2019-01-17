package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/BenchmarkToken"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/harness"
	. "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCreateGazillionTransactionsWhileTransportIsDuplicatingRandomMessages(t *testing.T) {
	ctrlRand := test.NewControlledRand(t)
	harness.Network(t).
		AllowingErrors(
			"error adding forwarded transaction to pending pool", // because we duplicate, among other messages, the transaction propagation message
			"ValidateBlockProposal blockHash mismatch",
			//"all consensus \\d* algos refused to validate the block",             //TODO(v1) investigate and explain, or fix and remove expected error
			//"FORK!! block already in storage, transaction block header mismatch", //TODO(v1) investigate and explain, or fix and remove expected error
		).
		//WithLogFilters(log.IgnoreMessagesMatching("leader failed to validate vote"),
		//	log.IgnoreErrorsMatching("transaction rejected: TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING")).
		Start(func(ctx context.Context, network harness.TestNetworkDriver) {

			//network.TransportTamperer().Duplicate(AnyNthMessage(7))

			sendTransfersAndAssertTotalBalance(ctx, network, t, 100, ctrlRand)
		})
}

func TestCreateGazillionTransactionsWhileTransportIsDroppingRandomMessages(t *testing.T) {
	ctrlRand := test.NewControlledRand(t)
	harness.Network(t).
		AllowingErrors(
			"all consensus \\d* algos refused to validate the block", //TODO(v1) investigate and explain, or fix and remove expected error
		).
		WithLogFilters(log.IgnoreMessagesMatching("leader failed to validate vote"), log.IgnoreErrorsMatching("transport failed to send")).
		WithNumNodes(4).Start(func(ctx context.Context, network harness.TestNetworkDriver) {

		network.TransportTamperer().Fail(HasHeader(ABenchmarkConsensusMessage).And(AnyNthMessage(7)))

		sendTransfersAndAssertTotalBalance(ctx, network, t, 100, ctrlRand)
	})
}

func TestCreateGazillionTransactionsWhileTransportIsDelayingRandomMessages(t *testing.T) {
	ctrlRand := test.NewControlledRand(t)
	harness.Network(t).
		AllowingErrors(
			"all consensus \\d* algos refused to validate the block", //TODO(v1) investigate and explain, or fix and remove expected error
		).
		WithLogFilters(log.IgnoreMessagesMatching("leader failed to validate vote")).
		WithNumNodes(4).Start(func(ctx context.Context, network harness.TestNetworkDriver) {

		network.TransportTamperer().Delay(func() time.Duration {
			return (time.Duration(ctrlRand.Intn(1000)) + 1000) * time.Microsecond // delay each message between 1000 and 2000 millis
		}, AnyNthMessage(2))

		sendTransfersAndAssertTotalBalance(ctx, network, t, 100, ctrlRand)
	})
}

func TestCreateGazillionTransactionsWhileTransportIsCorruptingRandomMessages(t *testing.T) {
	ctrlRand := test.NewControlledRand(t)
	harness.Network(t).WithNumNodes(4).Start(func(ctx context.Context, network harness.TestNetworkDriver) {
		t.Skip("this test causes the system to hang, seems like consensus algo stops")

		tamper := network.TransportTamperer().Corrupt(Not(HasHeader(ATransactionRelayMessage)).And(AnyNthMessage(7)), ctrlRand)

		sendTransfersAndAssertTotalBalance(ctx, network, t, 90, ctrlRand)

		tamper.Release(ctx)

		// assert that the system recovered properly
		sendTransfersAndAssertTotalBalance(ctx, network, t, 10, ctrlRand)

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

func sendTransfersAndAssertTotalBalance(ctx context.Context, network harness.TestNetworkDriver, t *testing.T, numTransactions int, ctrlRand *test.ControlledRand) {
	fromAddress := 5
	toAddress := 6
	contract := network.BenchmarkTokenContract()

	var expectedSum uint64 = 0
	var txHashes []primitives.Sha256
	for i := 0; i < numTransactions; i++ {
		amount := uint64(ctrlRand.Int63n(100))
		expectedSum += amount

		txHash := contract.TransferInBackground(ctx, ctrlRand.Intn(network.Size()), amount, fromAddress, toAddress)
		txHashes = append(txHashes, txHash)
	}
	for _, txHash := range txHashes {
		network.WaitForTransactionInState(ctx, txHash)
	}

	for i := 0; i < network.Size(); i++ {
		actualSum := contract.GetBalance(ctx, i, toAddress)
		require.EqualValuesf(t, expectedSum, actualSum, "recipient balance did not equal expected balance in node %d", i)

		actualRemainder := contract.GetBalance(ctx, i, fromAddress)
		require.EqualValuesf(t, benchmarktoken.TOTAL_SUPPLY-expectedSum, actualRemainder, "sender balance did not equal expected balance in node %d", i)
	}
}
