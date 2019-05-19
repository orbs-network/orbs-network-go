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
	newHarness().Start(b, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
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
	newHarness().WithConfigOverride(func(cfg config.OverridableConfig) config.OverridableConfig {
		c, err := cfg.MergeWithFileConfig(`{
	"transaction-pool-propagation-batch-size": 1000
}`)
		require.NoError(b, err)
		return c
	}).Start(b, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
		for n := 0; n < b.N; n++ {
			b.StartTimer()
			transferDuration, waitDuration := sendTransfersAndAssertTotalBalance(ctx, network, t, numTransactions, rnd)
			b.StopTimer()
			fmt.Println("finished sending ", numTransactions, "transactions in", transferDuration)
			fmt.Println("finished waiting for", numTransactions, "transactions in", waitDuration)
		}
	})
}
