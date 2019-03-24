// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package _manual

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"
)

var globalBlock *protocol.BlockPairContainer

func TestMemoryLeaks_AfterSomeTransactions(t *testing.T) {
	harness.Network(t).Start(func(ctx context.Context, network harness.NetworkDriver) {
		network.DeployBenchmarkToken(ctx, 5)
		globalBlock = nil

		t.Log("testing", network.Description()) // leader is nodeIndex 0, validator is nodeIndex 1

		for i := 0; i < 10; i++ {
			sendTransactionAndWaitUntilInState(ctx, network)
		}

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
		for i := 0; i < 500; i++ {
			sendTransactionAndWaitUntilInState(ctx, network)
		}

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

	harness.Network(t).Start(func(ctx context.Context, network harness.NetworkDriver) {
		network.DeployBenchmarkToken(ctx, 5)
		t.Log("testing", network.Description()) // leader is nodeIndex 0, validator is nodeIndex 1
		for i := 0; i < 20; i++ {
			sendTransactionAndWaitUntilInState(ctx, network)
		}
	})

	time.Sleep(20 * time.Millisecond) // give goroutines time to terminate

	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.GC()
	pprof.WriteHeapProfile(after)
}

func sendTransactionAndWaitUntilInState(ctx context.Context, network harness.NetworkDriver) {
	tx := network.Transfer(ctx, 0, 1, 5, 6)
	network.WaitForTransactionInState(ctx, 0, tx.TransactionReceipt().Txhash())
	network.WaitForTransactionInState(ctx, 1, tx.TransactionReceipt().Txhash())
}
