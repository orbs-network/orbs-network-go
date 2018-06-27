package acceptance

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/orbs-network/orbs-network-go/types"
	"github.com/orbs-network/orbs-network-go/testharness"
)

var _ = Describe("a leader node", func() {

	It("must get validations by all nodes to commit a transaction", func(done Done) {
		network := testharness.CreateTestNetwork()

		network.Gossip.FailConsensusRequests()
		network.Leader.SendTransaction(&types.Transaction{Value: 17})

		network.LeaderEvents.WaitForConsensusRounds(1)

		Expect(network.Leader.CallMethod()).To(Equal(0))
		Expect(network.Validator.CallMethod()).To(Equal(0))

		network.Gossip.PassConsensusRequests()
		network.LeaderBp.WaitForBlocks(1)
		Expect(network.Leader.CallMethod()).To(Equal(17))
		network.ValidatorBp.WaitForBlocks(1)
		Expect(network.Validator.CallMethod()).To(Equal(17))

		close(done)
	}, 1)
})
