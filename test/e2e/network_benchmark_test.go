package e2e

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
	"sync"
	"testing"
	"time"
)

func TestE2EStress(t *testing.T) {
	h := newHarness()
	ctrlRand := test.NewControlledRand(t)

	config := getConfig().stressTest

	if !config.enabled {
		t.Skip("Skipping stress test")
	}

	var wg sync.WaitGroup

	limiter := rate.NewLimiter(1000, 50)

	for i := int64(0); i < config.numberOfTransactions; i++ {
		if err := limiter.Wait(context.Background()); err == nil {
			wg.Add(1)

			go func() {
				defer wg.Done()

				targetKey, _ := keys.GenerateEd25519Key()
				targetAddress := builders.AddressFor(targetKey)
				amount := uint64(ctrlRand.Intn(10))

				_, _, err2 := h.sendTransaction(OwnerOfAllSupply, "BenchmarkToken", "transfer", uint64(amount), []byte(targetAddress))

				if err2 != nil {
					t.Logf("error sending transaction %s\n", err)
				}
			}()
		} else {
			t.Logf("error %s\n", err)
		}
	}

	wg.Wait()

	var m metrics

	require.True(t, test.Eventually(1*time.Minute, func() bool {
		m = h.getMetrics()
		return m != nil
	}), "could not retrieve metrics")

	txCount := m["TransactionPool.CommittedPool.TransactionCount"]["Value"].(float64)

	expectedNumberOfTx := float64((100 - config.acceptableFailureRate) / 100 * config.numberOfTransactions)

	require.Condition(t, func() (success bool) {
		return txCount >= expectedNumberOfTx
	}, "transaction processed (%f) < expected transactions processed (%f) out of %i transactions sent", txCount, expectedNumberOfTx, config.numberOfTransactions)

	ratePerSecond := m["TransactionPool.RatePerSecond"]["Rate"].(float64)

	require.Condition(t, func() (success bool) {
		return ratePerSecond >= config.targetTPS
	}, "actual tps (%f) is less than target tps (%f)", ratePerSecond, config.targetTPS)
}
