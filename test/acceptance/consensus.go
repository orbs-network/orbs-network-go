package acceptance

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-network-go/types"
)

var _ = Describe("a leader node", func() {

	It("must get validations by all nodes to commit a transaction", func(done Done) {
		network := harness.CreateTestNetwork()
		defer network.FlushLog()
		consensusRound := network.LeaderLoopControl().LatchFor("consensus_round")

		consensusRound.Brake()
		network.Gossip().FailConsensusRequests()

		<- network.SendTransaction(network.Leader(), &types.Transaction{Value: 17})

		consensusRound.Tick()

		Expect(<- network.CallMethod(network.Leader())).To(Equal(0))
		Expect(<- network.CallMethod(network.Validator())).To(Equal(0))

		network.Gossip().PassConsensusRequests()

		consensusRound.Release()

		network.LeaderBp().WaitForBlocks(1)
		Expect(<- network.CallMethod(network.Leader())).To(Equal(17))

		network.ValidatorBp().WaitForBlocks(1)
		Expect(<- network.CallMethod(network.Validator())).To(Equal(17))


		close(done)
	}, 1)
})
