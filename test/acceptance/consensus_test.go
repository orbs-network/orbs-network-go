package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLeanHelixLeaderGetsValidationsBeforeCommit(t *testing.T) {
	t.Skip("putting lean helix on hold until external library is integrated")
	harness.Network(t).WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).Start(func(ctx context.Context, network harness.InProcessTestNetwork) {

		network.DeployBenchmarkToken(ctx, 5)

		prePrepareLatch := network.GossipTransport().LatchOn(adapter.LeanHelixMessage(consensus.LEAN_HELIX_PRE_PREPARE))
		prePrepareTamper := network.GossipTransport().Fail(adapter.LeanHelixMessage(consensus.LEAN_HELIX_PRE_PREPARE))
		<-network.SendTransfer(ctx, 0, 17, 5, 6)

		prePrepareLatch.Wait()
		require.EqualValues(t, 0, <-network.CallGetBalance(ctx, 0, 6), "initial getBalance result on leader")
		require.EqualValues(t, 0, <-network.CallGetBalance(ctx, 1, 6), "initial getBalance result on non leader")

		prePrepareTamper.Release(ctx)
		prePrepareLatch.Remove()

		if err := network.BlockPersistence(0).GetBlockTracker().WaitForBlock(ctx, 1); err != nil {
			t.Errorf("waiting for block on node 0 failed: %s", err)
		}
		require.EqualValues(t, 17, <-network.CallGetBalance(ctx, 0, 6), "eventual getBalance result on leader")

		if err := network.BlockPersistence(1).GetBlockTracker().WaitForBlock(ctx, 1); err != nil {
			t.Errorf("waiting for block on node 1 failed: %s", err)
		}
		require.EqualValues(t, 17, <-network.CallGetBalance(ctx, 1, 6), "eventual getBalance result on non leader")

	})
}

func TestBenchmarkConsensusLeaderGetsVotesBeforeNextBlock(t *testing.T) {
	harness.Network(t).
		AllowingErrors(
			"consensus round tick failed", // (aborting shared state update due to inconsistency) //TODO investigate and explain, or fix and remove expected error
		).WithMaxTxPerBlock(1).
		AllowingErrors(
			"consensus round tick failed", // (aborting shared state update due to inconsistency) //TODO investigate and explain, or fix and remove expected error
		).Start(func(ctx context.Context, network harness.InProcessTestNetwork) {

		network.DeployBenchmarkToken(ctx, 5)

		committedTamper := network.GossipTransport().Fail(adapter.BenchmarkConsensusMessage(consensus.BENCHMARK_CONSENSUS_COMMITTED))
		blockSyncTamper := network.GossipTransport().Fail(adapter.BlockSyncMessage(gossipmessages.BLOCK_SYNC_AVAILABILITY_REQUEST)) // block sync discovery message so it does not add the blocks in a 'back door'
		committedLatch := network.GossipTransport().LatchOn(adapter.BenchmarkConsensusMessage(consensus.BENCHMARK_CONSENSUS_COMMITTED))

		network.SendTransferInBackground(ctx, 0, 0, 5, 6) // send a transaction so that network advances to block 1. the tamper prevents COMMITTED messages from reaching leader, so it doesn't move to block 2
		committedLatch.Wait()                             // wait for validator to try acknowledge that it reached block 1 (and fail)
		committedLatch.Wait()                             // wait for another consensus round (to make sure transaction(0) does not arrive after transaction(17) due to scheduling flakiness

		txHash := network.SendTransferInBackground(ctx, 0, 17, 5, 6) // this should be included in block 2 which will not be closed until leader knows network is at block 2

		committedLatch.Wait()

		require.EqualValues(t, 0, <-network.CallGetBalance(ctx, 0, 6), "initial getBalance result on leader")
		require.EqualValues(t, 0, <-network.CallGetBalance(ctx, 1, 6), "initial getBalance result on non leader")

		committedLatch.Remove()
		committedTamper.Release(ctx) // this will allow COMMITTED messages to reach leader so that it can progress

		network.WaitForTransactionInState(ctx, 0, txHash)
		require.EqualValues(t, 17, <-network.CallGetBalance(ctx, 0, 6), "eventual getBalance result on leader")

		network.WaitForTransactionInState(ctx, 1, txHash)
		require.EqualValues(t, 17, <-network.CallGetBalance(ctx, 1, 6), "eventual getBalance result on non leader")

		blockSyncTamper.Release(ctx)
	})
}
