package e2e

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestE2EStress(t *testing.T) {
	h := newHarness()
	defer h.gracefulShutdown()

	config := getConfig().stressTest

	if !config.enabled {
		t.Skip("Skipping stress test")
	}

	var wg sync.WaitGroup

	limiter := rate.NewLimiter(1000, 50)

	for i := int64(0); i < config.numberOfTransactions; i++ {
		if err := limiter.Wait(context.TODO()); err == nil {
			wg.Add(1)

			go func() {
				defer wg.Done()

				signerKeyPair := keys.Ed25519KeyPairForTests(5)
				targetAddress := builders.AddressForEd25519SignerForTests(6)
				amount := uint64(rand.Intn(10))

				transfer := builders.TransferTransaction().WithEd25519Signer(signerKeyPair).WithAmountAndTargetAddress(amount, targetAddress).Builder()
				_, err := h.sendTransaction(transfer)

				if err != nil {
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
