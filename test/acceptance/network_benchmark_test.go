package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"golang.org/x/time/rate"
	"sync"
	"testing"
)

func BenchmarkInMemoryNetwork(b *testing.B) {
	var wg sync.WaitGroup

	limiter := rate.NewLimiter(1000, 100)
	ctrlRand := test.NewControlledRand(b)

	harness.Network(b).
		WithLogFilters(log.DiscardAll()).
		WithNumNodes(3).Start(func(ctx context.Context, network harness.TestNetworkDriver) {

		contract := network.BenchmarkTokenContract()

		contract.DeployBenchmarkToken(ctx, 5)

		for i := 0; i < b.N; i++ {
			if err := limiter.Wait(ctx); err == nil {
				wg.Add(1)

				go func() {
					defer wg.Done()
					nodeNum := ctrlRand.Intn(network.Size())
					contract.Transfer(ctx, nodeNum, uint64(ctrlRand.Intn(i)), 5, 6)
				}()

			}

		}

		wg.Wait()

	})
}
