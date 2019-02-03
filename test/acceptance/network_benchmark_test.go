package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"golang.org/x/time/rate"
	"sync"
	"testing"
)

func BenchmarkInMemoryNetwork(b *testing.B) {
	var wg sync.WaitGroup

	limiter := rate.NewLimiter(1000, 100)
	ctrlRand := test.NewControlledRand(b)

	newHarness(b).
		WithLogFilters(log.DiscardAll()).
		WithNumNodes(4).Start(func(ctx context.Context, network NetworkHarness) {

		contract := network.DeployBenchmarkTokenContract(ctx, 5)

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
