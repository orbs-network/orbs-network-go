package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"golang.org/x/time/rate"
	"math/rand"
	"sync"
	"testing"
)

func BenchmarkInMemoryNetwork(b *testing.B) {

	var wg sync.WaitGroup

	limiter := rate.NewLimiter(1000, 100)

	harness.Network(b).
		WithLogFilters(log.DiscardAll()).
		WithNumNodes(3).Start(func(ctx context.Context, network harness.TestNetworkDriver) {

		contract := network.GetBenchmarkTokenContract()

		contract.DeployBenchmarkToken(ctx, 5)

		for i := 0; i < b.N; i++ {
			if err := limiter.Wait(context.TODO()); err == nil {
				wg.Add(1)

				go func() {
					defer wg.Done()
					nodeNum := rand.Intn(network.Size())
					<-contract.SendTransfer(ctx, nodeNum, uint64(rand.Intn(i)), 5, 6)
				}()

			}

		}

		wg.Wait()

	})

}
