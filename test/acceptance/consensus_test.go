package acceptance

import (
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLeanHelixLeaderGetsValidationsBeforeCommit(t *testing.T) {
	harness.WithNetwork(t, 2, harness.WithAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX), func(network harness.AcceptanceTestNetwork) {

		network.DeployBenchmarkToken()

		prePrepareLatch := network.GossipTransport().LatchOn(adapter.LeanHelixMessage(consensus.LEAN_HELIX_PRE_PREPARE))
		prePrepareTamper := network.GossipTransport().Fail(adapter.LeanHelixMessage(consensus.LEAN_HELIX_PRE_PREPARE))
		<-network.SendTransfer(0, 17)

		prePrepareLatch.Wait()
		require.EqualValues(t, 0, <-network.CallGetBalance(0), "initial getBalance result on leader")
		require.EqualValues(t, 0, <-network.CallGetBalance(1), "initial getBalance result on non leader")

		prePrepareTamper.Release()
		prePrepareLatch.Remove()

		network.BlockPersistence(0).WaitForBlocks(1)
		require.EqualValues(t, 17, <-network.CallGetBalance(0), "eventual getBalance result on leader")

		network.BlockPersistence(1).WaitForBlocks(1)
		require.EqualValues(t, 17, <-network.CallGetBalance(1), "eventual getBalance result on non leader")

	})
}

func TestBenchmarkConsensusLeaderGetsVotesBeforeNextBlock(t *testing.T) {
	harness.WithNetwork(t,2, harness.WithAlgos(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS), func(network harness.AcceptanceTestNetwork) {

		network.DeployBenchmarkToken()

		committedLatch := network.GossipTransport().LatchOn(adapter.BenchmarkConsensusMessage(consensus.BENCHMARK_CONSENSUS_COMMITTED))
		committedTamper := network.GossipTransport().Fail(adapter.BenchmarkConsensusMessage(consensus.BENCHMARK_CONSENSUS_COMMITTED))
		<-network.SendTransfer(0, 0)

		committedLatch.Wait()
		<-network.SendTransfer(0, 17)

		committedLatch.Wait()
		require.EqualValues(t, 0, <-network.CallGetBalance(0), "initial getBalance result on leader")
		require.EqualValues(t, 0, <-network.CallGetBalance(1), "initial getBalance result on non leader")

		committedTamper.Release()
		committedLatch.Remove()

		network.BlockPersistence(0).WaitForBlocks(2)
		require.EqualValues(t, 17, <-network.CallGetBalance(0), "eventual getBalance result on leader")

		network.BlockPersistence(1).WaitForBlocks(2)
		require.EqualValues(t, 17, <-network.CallGetBalance(1), "eventual getBalance result on non leader")

	})
}
