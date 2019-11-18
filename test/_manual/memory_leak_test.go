// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package _manual

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/acceptance"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"
)

var globalBlock *protocol.BlockPairContainer

func TestMemoryLeaks_AfterSomeTransactions(t *testing.T) {
	acceptance.
		NewHarness().
		WithTestTimeout(5*time.Minute).
		Start(t, func(t testing.TB, ctx context.Context, network *acceptance.Network) {

			benchmarkTokenClient := network.DeployBenchmarkTokenContract(ctx, 5)
			globalBlock = nil

			sendTransactionsSequentially(ctx, network, benchmarkTokenClient, 10)

			runtime.MemProfileRate = 100
			before, _ := os.Create("/tmp/mem-tx-before.prof")
			defer before.Close()
			after, _ := os.Create("/tmp/mem-tx-after.prof")
			defer after.Close()

			runtime.GC()
			runtime.GC()
			runtime.GC()
			runtime.GC()
			pprof.WriteHeapProfile(before)

			// play with these lines to find memory leaks in closed blocks
			sendTransactionsSequentially(ctx, network, benchmarkTokenClient, 500)

			// play with these lines to see a leak example
			//globalBlock = builders.BlockPair().WithTransactions(10000).Build()
			//globalBlock = nil

			runtime.GC()
			runtime.GC()
			runtime.GC()
			runtime.GC()
			pprof.WriteHeapProfile(after)
		})
}

func TestMemoryLeaks_OnSystemShutdown(t *testing.T) {
	runtime.MemProfileRate = 1
	before, _ := os.Create("/tmp/mem-shutdown-before.prof")
	defer before.Close()
	after, _ := os.Create("/tmp/mem-shutdown-after.prof")
	defer after.Close()

	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.GC()
	pprof.WriteHeapProfile(before)

	acceptance.
		NewHarness().
		WithTestTimeout(5*time.Minute).
		Start(t, func(t testing.TB, ctx context.Context, network *acceptance.Network) {
			benchmarkTokenClient := network.DeployBenchmarkTokenContract(ctx, 5)

			sendTransactionsSequentially(ctx, network, benchmarkTokenClient, 20)
		})

	time.Sleep(20 * time.Millisecond) // give goroutines time to terminate

	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.GC()
	pprof.WriteHeapProfile(after)
}

func sendTransactionsSequentially(ctx context.Context, network *acceptance.Network, client callcontract.BenchmarkTokenClient, txCount int) {
	for i := 0; i < txCount; i++ {
		_, txHash := client.Transfer(ctx, 0, 1, 5, 6)
		network.WaitForTransactionInState(ctx, txHash)
	}
}
