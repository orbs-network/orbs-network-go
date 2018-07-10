package acceptance

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-network-go/test/harness"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
)

var _ = Describe("a leader node", func() {

	It("must get validations by all nodes to commit a transaction", func(done Done) {

		network := harness.CreateTestNetwork()
		defer network.FlushLog()
		consensusRound := network.LeaderLoopControl().LatchFor("consensus_round")

		consensusRound.Brake()
		network.Gossip().Fail(gossipAdapter.PrePrepareMessage)

		<- network.Transfer(network.Leader(), 17)

		consensusRound.Tick()

		Expect(<- network.GetBalance(network.Leader())).To(BeEquivalentTo(0))
		Expect(<- network.GetBalance(network.Validator())).To(BeEquivalentTo(0))

		network.Gossip().Pass(gossipAdapter.PrePrepareMessage)

		consensusRound.Release()

		network.LeaderBp().WaitForBlocks(1)
		Expect(<- network.GetBalance(network.Leader())).To(BeEquivalentTo(17))

		network.ValidatorBp().WaitForBlocks(1)
		Expect(<- network.GetBalance(network.Validator())).To(BeEquivalentTo(17))

		close(done)
	}, 1)
})
