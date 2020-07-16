package acceptance

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/stretchr/testify/require"
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
	NewHarness().WithConfigOverride(func(cfg config.OverridableConfig) config.OverridableConfig {
		c, err := cfg.MergeWithFileConfig(`{
	"consensus-context-maximum-transactions-in-block": 70,
	"transaction-pool-propagation-batch-size": 100
}`)
		require.NoError(b, err)
		return c
	}).Start(b, func(t testing.TB, ctx context.Context, network *Network) {
		for n := 0; n < b.N; n++ {
			b.StartTimer()
			transferDuration, waitDuration := sendTransfersAndAssertTotalBalance(ctx, network, t, numTransactions, rnd)
			b.StopTimer()
			fmt.Println("finished sending ", numTransactions, "transactions in", transferDuration)
			fmt.Println("finished waiting for", numTransactions, "transactions in", waitDuration)
		}
	})
}
