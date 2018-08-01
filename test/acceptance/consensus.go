package acceptance

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
)

var _ = Describe("a leader node", func() {

	It("must get validations by all nodes to commit a transaction", func(done Done) {
		consensusAlgos := []consensus.ConsensusAlgoType{consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX}
		harness.WithNetwork(2, consensusAlgos, func(ctx context.Context, network harness.AcceptanceTestNetwork) {

			prePrepareLatch := network.GossipTransport().LatchOn(adapter.LeanHelixMessage(consensus.LEAN_HELIX_PRE_PREPARE))
			prePrepareTamper := network.GossipTransport().Fail(adapter.LeanHelixMessage(consensus.LEAN_HELIX_PRE_PREPARE))
			<-network.SendTransfer(0, 17)

			prePrepareLatch.Wait()
			Expect(<-network.CallGetBalance(0)).To(BeEquivalentTo(0))
			Expect(<-network.CallGetBalance(1)).To(BeEquivalentTo(0))

			prePrepareTamper.Release()
			prePrepareLatch.Remove()

			network.BlockPersistence(0).WaitForBlocks(1)
			Expect(<-network.CallGetBalance(0)).To(BeEquivalentTo(17))

			network.BlockPersistence(1).WaitForBlocks(1)
			Expect(<-network.CallGetBalance(1)).To(BeEquivalentTo(17))

		})

		close(done)

	}, 1)

	It("must get validations by all nodes to commit a transaction", func(done Done) {
		consensusAlgos := []consensus.ConsensusAlgoType{consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS}
		harness.WithNetwork(2, consensusAlgos, func(ctx context.Context, network harness.AcceptanceTestNetwork) {

			committedLatch := network.GossipTransport().LatchOn(adapter.BenchmarkConsensusMessage(consensus.BENCHMARK_CONSENSUS_COMMITTED))
			committedTamper := network.GossipTransport().Fail(adapter.BenchmarkConsensusMessage(consensus.BENCHMARK_CONSENSUS_COMMITTED))
			<-network.SendTransfer(0, 17)

			committedLatch.Wait()
			Expect(<-network.CallGetBalance(0)).To(BeEquivalentTo(0))
			Expect(<-network.CallGetBalance(1)).To(BeEquivalentTo(0))

			committedTamper.Release()
			committedLatch.Remove()

			network.BlockPersistence(0).WaitForBlocks(1)
			Expect(<-network.CallGetBalance(0)).To(BeEquivalentTo(17))

			network.BlockPersistence(1).WaitForBlocks(1)
			Expect(<-network.CallGetBalance(1)).To(BeEquivalentTo(17))

		})

		close(done)

	}, 1)

})
