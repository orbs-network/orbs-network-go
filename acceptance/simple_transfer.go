package acceptance

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/orbs-network/orbs-network-go/types"
	"github.com/orbs-network/orbs-network-go/testharness"
)

var _ = Describe("a leader node", func() {

	It("commits transactions to all nodes", func() {
		network := testharness.CreateTestNetwork()

		network.Leader.SendTransaction(&types.Transaction{Value: 17})
		network.Leader.SendTransaction(&types.Transaction{Value: 97, Invalid: true})
		network.Leader.SendTransaction(&types.Transaction{Value: 22})

		network.LeaderBp.WaitForBlocks(2)
		Expect(network.Leader.CallMethod()).To(Equal(39))

		network.ValidatorBp.WaitForBlocks(2)
		Expect(network.Validator.CallMethod()).To(Equal(39))
	})
})

var _ = Describe("a non-leader (validator) node", func() {

	It("propagates transactions to leader but does not commit them itself", func() {
		network := testharness.CreateTestNetwork()

		network.Gossip.PauseForwards()
		network.Validator.SendTransaction(&types.Transaction{Value: 17})

		Expect(network.Leader.CallMethod()).To(Equal(0))
		Expect(network.Validator.CallMethod()).To(Equal(0))

		network.Gossip.ResumeForwards()
		network.LeaderBp.WaitForBlocks(1)
		Expect(network.Leader.CallMethod()).To(Equal(17))
		network.ValidatorBp.WaitForBlocks(1)
		Expect(network.Validator.CallMethod()).To(Equal(17))
	})

})
