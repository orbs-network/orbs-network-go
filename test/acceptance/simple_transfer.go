package acceptance

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-network-go/gossip"
)

var _ = Describe("a leader node", func() {

	It("commits transactions to all nodes", func() {
		network := harness.CreateTestNetwork()

		network.Transfer(network.Leader(), 17)
		network.Transfer(network.Leader(), 1000000)
		network.Transfer(network.Leader(), 22)

		network.LeaderBp().WaitForBlocks(2)
		Expect(<- network.GetBalance(network.Leader())).To(BeEquivalentTo(39))

		network.ValidatorBp().WaitForBlocks(2)
		Expect(<- network.GetBalance(network.Validator())).To(BeEquivalentTo(39))

	})
})

var _ = Describe("a non-leader (validator) node", func() {

	It("propagates transactions to leader but does not commit them itself", func() {

		network := harness.CreateTestNetwork()

		network.Gossip().Pause(gossip.ForwardTransactionMessage)
		network.Transfer(network.Leader(), 17)

		Expect(<- network.GetBalance(network.Leader())).To(BeEquivalentTo(0))
		Expect(<- network.GetBalance(network.Validator())).To(BeEquivalentTo(0))

		network.Gossip().Resume(gossip.ForwardTransactionMessage)
		network.LeaderBp().WaitForBlocks(1)
		Expect(<- network.GetBalance(network.Leader())).To(BeEquivalentTo(17))
		network.ValidatorBp().WaitForBlocks(1)
		Expect(<- network.GetBalance(network.Validator())).To(BeEquivalentTo(17))
	})

})
