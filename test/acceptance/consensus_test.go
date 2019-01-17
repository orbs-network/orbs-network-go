package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/internodesync"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/testkit"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TODO v1 Rewrite this test without knowing specific LH message types
// Add more nodes for consensus to work (min 4)
func TestLeanHelixLeaderGetsValidationsBeforeCommit(t *testing.T) {
	t.Skipf("Change this - Orbs is not supposed to know LH message types")
	//harness.
	//	newHarness(t).
	//	WithNumNodes(4).
	//	WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
	//	Start(func(ctx context.Context, network NetworkHarness) {
	//
	//		contract := network.BenchmarkTokenContract()
	//
	//		amount := uint64(17)
	//		fromAddress := 5
	//		toAddress := 6
	//		leaderIndex := 0
	//		validatorIndex := 1
	//
	//		contract.DeployBenchmarkToken(ctx, fromAddress)
	//
	//		// these get preds
	//		// reimpl this, it is supposed to know the ppm but
	//
	//		prePrepareLatch := network.TransportTamperer().LatchOn(adapter.LeanHelixMessage(leanhelix.LEAN_HELIX_PREPREPARE))
	//		prePrepareTamper := network.TransportTamperer().Fail(adapter.LeanHelixMessage(leanhelix.LEAN_HELIX_PREPREPARE))
	//
	//		contract.Transfer(ctx, leaderIndex, amount, fromAddress, toAddress)
	//
	//		prePrepareLatch.Wait() // blocking
	//		require.EqualValues(t, 0, contract.GetBalance(ctx, leaderIndex, toAddress), "initial getBalance result on leader")
	//		require.EqualValues(t, 0, contract.GetBalance(ctx, validatorIndex, toAddress), "initial getBalance result on non leader")
	//
	//		prePrepareTamper.Release(ctx)
	//		prePrepareLatch.Remove()
	//
	//		if err := network.BlockPersistence(leaderIndex).GetBlockTracker().WaitForBlock(ctx, 1); err != nil {
	//			t.Errorf("waiting for block on node 0 failed: %s", err)
	//		}
	//		require.EqualValues(t, amount, contract.GetBalance(ctx, leaderIndex, toAddress), "eventual getBalance result on leader")
	//
	//		if err := network.BlockPersistence(validatorIndex).GetBlockTracker().WaitForBlock(ctx, 1); err != nil {
	//			t.Errorf("waiting for block on node 1 failed: %s", err)
	//		}
	//		require.EqualValues(t, amount, contract.GetBalance(ctx, validatorIndex, toAddress), "eventual getBalance result on non leader")
	//
	//	})
}

// LH: Not relevant
func TestBenchmarkConsensusLeaderGetsVotesBeforeNextBlock(t *testing.T) {
	newHarness(t).
		WithLogFilters(log.ExcludeField(internodesync.LogTag), log.ExcludeEntryPoint("BlockSync")).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS).
		WithMaxTxPerBlock(1).
		Start(func(parent context.Context, network NetworkHarness) {
			ctx, cancel := context.WithTimeout(parent, 1*time.Second)
			defer cancel()

			contract := network.BenchmarkTokenContract()
			contract.DeployBenchmarkToken(ctx, 5)

			committedTamper := network.TransportTamperer().Fail(testkit.BenchmarkConsensusMessage(consensus.BENCHMARK_CONSENSUS_COMMITTED))
			blockSyncTamper := network.TransportTamperer().Fail(testkit.BlockSyncMessage(gossipmessages.BLOCK_SYNC_AVAILABILITY_REQUEST)) // block sync discovery message so it does not add the blocks in a 'back door'
			committedLatch := network.TransportTamperer().LatchOn(testkit.BenchmarkConsensusMessage(consensus.BENCHMARK_CONSENSUS_COMMITTED))

			contract.TransferInBackground(ctx, 0, 0, 5, 6) // send a transaction so that network advances to block 1. the tamper prevents COMMITTED messages from reaching leader, so it doesn't move to block 2
			committedLatch.Wait()                          // wait for validator to try acknowledge that it reached block 1 (and fail)
			committedLatch.Wait()                          // wait for another consensus round (to make sure transaction(0) does not arrive after transaction(17) due to scheduling flakiness

			txHash := contract.TransferInBackground(ctx, 0, 17, 5, 6) // this should be included in block 2 which will not be closed until leader knows network is at block 2

			committedLatch.Wait()

			require.EqualValues(t, 0, contract.GetBalance(ctx, 0, 6), "initial getBalance result on leader")
			require.EqualValues(t, 0, contract.GetBalance(ctx, 1, 6), "initial getBalance result on non leader")

			committedLatch.Remove()
			committedTamper.Release(ctx) // this will allow COMMITTED messages to reach leader so that it can progress

			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 17, contract.GetBalance(ctx, 0, 6), "eventual getBalance result on leader")

			network.WaitForTransactionInNodeState(ctx, txHash, 1)
			require.EqualValues(t, 17, contract.GetBalance(ctx, 1, 6), "eventual getBalance result on non leader")

			blockSyncTamper.Release(ctx)
		})
}
