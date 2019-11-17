package acceptance

import (
	"context"
	"encoding/json"
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

type EssentialMetricsHistogram struct {
	Min     float64
	P50     float64
	P95     float64
	P99     float64
	Max     float64
	Avg     float64
	Samples int64
}

type EssentialMetrics struct {
	BlockHeight              uint64
	TxTimeSpentInQueueMillis EssentialMetricsHistogram
	TxCommitRatePerSecond    float64
}

func parseEssentialMetricsHistogram(value map[string]interface{}) EssentialMetricsHistogram {
	return EssentialMetricsHistogram{
		Min:     value["Min"].(float64),
		Max:     value["Max"].(float64),
		P50:     value["P50"].(float64),
		P95:     value["P95"].(float64),
		P99:     value["P99"].(float64),
		Avg:     value["Avg"].(float64),
		Samples: int64(value["Samples"].(float64)),
	}
}

func parseEssentialMetrics(info interface{}) EssentialMetrics {
	raw, _ := json.Marshal(info)
	m := make(map[string]map[string]interface{})
	json.Unmarshal(raw, &m)

	e := EssentialMetrics{}

	for key, value := range m {
		switch key {
		case "BlockStorage.BlockHeight":
			e.BlockHeight = uint64(value["Value"].(float64))
		case "TransactionPool.PendingPool.TimeSpentInQueue.Millis":
			e.TxTimeSpentInQueueMillis = parseEssentialMetricsHistogram(value)
		case "TransactionPool.CommitRate.PerSecond":
			e.TxCommitRatePerSecond = value["Rate"].(float64)
		}
	}

	return e
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

		e := parseEssentialMetrics(network.MetricRegistry(0).ExportAll())
		fmt.Println("stats", e)
	})
}
