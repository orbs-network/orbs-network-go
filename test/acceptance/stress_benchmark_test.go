package acceptance

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"testing"
)

func BenchmarkHappyFlow1k(b *testing.B) {
	numTransactions := 1000

	rnd := rand.NewControlledRand(b)
	NewHarness().Start(b, func(t testing.TB, ctx context.Context, network *Network) {
		for n := 0; n < b.N; n++ {
			b.StartTimer()
			transferDuration, waitDuration := sendTransfersAndAssertTotalBalance(ctx, network, t, numTransactions, rnd)
			b.StopTimer()

			fmt.Println("finished sending ", numTransactions, "transactions in", transferDuration)
			fmt.Println("finished waiting for", numTransactions, "transactions in", waitDuration)
		}
	})
}

func BenchmarkHappyFlow1kWithOverrides(b *testing.B) {
	numTransactions := 1000

	rnd := rand.NewControlledRand(b)
	NewHarness().
		WithConfigOverride(
			config.NodeConfigKeyValue{Key: config.CONSENSUS_CONTEXT_MAXIMUM_TRANSACTIONS_IN_BLOCK, Value: config.NodeConfigValue{Uint32Value: 70}},
			config.NodeConfigKeyValue{Key: config.TRANSACTION_POOL_PROPAGATION_BATCH_SIZE, Value: config.NodeConfigValue{Uint32Value: 100}},
		).
		Start(b, func(t testing.TB, ctx context.Context, network *Network) {
		for n := 0; n < b.N; n++ {
			b.StartTimer()
			transferDuration, waitDuration := sendTransfersAndAssertTotalBalance(ctx, network, t, numTransactions, rnd)
			b.StopTimer()
			fmt.Println("finished sending ", numTransactions, "transactions in", transferDuration)
			fmt.Println("finished waiting for", numTransactions, "transactions in", waitDuration)
		}
	})
}
