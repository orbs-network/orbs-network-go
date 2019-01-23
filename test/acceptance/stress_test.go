package acceptance

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	. "github.com/orbs-network/orbs-network-go/services/gossip/adapter/testkit"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/BenchmarkToken"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCreateGazillionTransactionsWhileTransportIsDuplicatingRandomMessages(t *testing.T) {
	rnd := test.NewControlledRand(t)
	newHarness(t).
		AllowingErrors(
			"error adding forwarded transaction to pending pool", // because we duplicate, among other messages, the transaction propagation message
		).
		Start(func(ctx context.Context, network NetworkHarness) {
			network.TransportTamperer().Duplicate(WithPercentChance(rnd, 30))
			sendTransfersAndAssertTotalBalance(ctx, network, t, 100, rnd)
		})
}

func TestCreateGazillionTransactionsWhileTransportIsDroppingRandomMessages(t *testing.T) {
	rnd := test.NewControlledRand(t)
	newHarness(t).
		Start(func(ctx context.Context, network NetworkHarness) {
			network.TransportTamperer().Fail(HasHeader(AConsensusMessage).And(WithPercentChance(rnd, 30)))
			sendTransfersAndAssertTotalBalance(ctx, network, t, 100, rnd)
		})
}

func TestCreateGazillionTransactionsWhileTransportIsDelayingRandomMessages(t *testing.T) {
	rnd := test.NewControlledRand(t)
	newHarness(t).
		Start(func(ctx context.Context, network NetworkHarness) {

			network.TransportTamperer().Delay(func() time.Duration {
				return (time.Duration(rnd.Intn(1000)) + 1000) * time.Microsecond // delay each message between 1000 and 2000 millis
			}, WithPercentChance(rnd, 50))

			sendTransfersAndAssertTotalBalance(ctx, network, t, 100, rnd)
		})
}

// TODO (v1) This should work - fix and remove Skip
func TestCreateGazillionTransactionsWhileTransportIsCorruptingRandomMessages(t *testing.T) {
	t.Skip("This should work - fix and remove Skip")
	rnd := test.NewControlledRand(t)
	newHarness(t).WithNumNodes(4).Start(func(ctx context.Context, network NetworkHarness) {
		tamper := network.TransportTamperer().Corrupt(Not(HasHeader(ATransactionRelayMessage)).And(WithPercentChance(rnd, 30)), rnd)
		sendTransfersAndAssertTotalBalance(ctx, network, t, 90, rnd)
		tamper.Release(ctx)

		// assert that the system recovered properly
		sendTransfersAndAssertTotalBalance(ctx, network, t, 10, rnd)

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

func WithPercentChance(ctrlRand *test.ControlledRand, pct int) MessagePredicate {
	var hit bool
	if pct >= 100 {
		hit = true
	} else if pct <= 0 {
		hit = false
	} else {
		hit = ctrlRand.Intn(101) <= pct
	}
	return func(data *adapter.TransportData) bool {
		return hit
	}
}

func TestWithNPctChance_AlwaysTrue(t *testing.T) {
	ctrlRand := test.NewControlledRand(t)
	require.True(t, WithPercentChance(ctrlRand, 100)(nil), "100% chance should always return true")
}

func TestWithNPctChance_AlwaysFalse(t *testing.T) {
	ctrlRand := test.NewControlledRand(t)
	require.False(t, WithPercentChance(ctrlRand, 0)(nil), "0% chance should always return false")
}

func TestWithNPctChance_ManualCheck(t *testing.T) {
	ctrlRand := test.NewControlledRand(t)
	tries := 1000
	pct := ctrlRand.Intn(100)
	hits := 0
	for i := 0; i < tries; i++ {
		if WithPercentChance(ctrlRand, pct)(nil) {
			hits++
		}
	}
	fmt.Printf("Manual test for WithPercentChance: Tries=%d Chance=%d%% Hits=%d\n", tries, pct, hits)
}

func sendTransfersAndAssertTotalBalance(ctx context.Context, network NetworkHarness, t *testing.T, numTransactions int, ctrlRand *test.ControlledRand) {
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
