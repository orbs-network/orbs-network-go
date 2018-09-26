package _manual

import (
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
	harness.Network(t).Start(func(network harness.InProcessNetwork) {
		network.DeployBenchmarkToken(5)
		globalBlock = nil

		t.Log("testing", network.Description()) // leader is nodeIndex 0, validator is nodeIndex 1

		for i := 0; i < 10; i++ {
			sendTransactionAndWaitUntilInState(network)
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
			sendTransactionAndWaitUntilInState(network)
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

	harness.Network(t).Start(func(network harness.InProcessNetwork) {
		network.DeployBenchmarkToken(5)
		t.Log("testing", network.Description()) // leader is nodeIndex 0, validator is nodeIndex 1
		for i := 0; i < 20; i++ {
			sendTransactionAndWaitUntilInState(network)
		}
	})

	time.Sleep(20 * time.Millisecond) // give goroutines time to terminate

	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.GC()
	pprof.WriteHeapProfile(after)
}

func sendTransactionAndWaitUntilInState(network harness.InProcessNetwork) {
	tx := <-network.SendTransfer(0, 1, 5, 6)
	network.WaitForTransactionInState(0, tx.TransactionReceipt().Txhash())
	network.WaitForTransactionInState(1, tx.TransactionReceipt().Txhash())
}
