package acceptance

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/orbs-network/orbs-network-go/types"
	"github.com/orbs-network/orbs-network-go/testharness"
	"github.com/orbs-network/orbs-network-go/events"
)

var _ = Describe("a leader node", func() {

	It("must get validations by all nodes to commit a transaction", func(done Done) {
		network := testharness.CreateTestNetwork()
		defer network.FlushLog()

		network.Gossip.FailConsensusRequests()
		network.Leader.GetPublicApi().SendTransaction(&types.Transaction{Value: 17})

		network.LeaderLatch.WaitFor(events.FinishedConsensusRound)

		Expect(network.Leader.GetPublicApi().CallMethod()).To(Equal(0))
		Expect(network.Validator.GetPublicApi().CallMethod()).To(Equal(0))

		network.Gossip.PassConsensusRequests()

		network.LeaderBp.WaitForBlocks(1)
		Expect(network.Leader.GetPublicApi().CallMethod()).To(Equal(17))
		network.ValidatorBp.WaitForBlocks(1)
		Expect(network.Validator.GetPublicApi().CallMethod()).To(Equal(17))

		close(done)
	}, 1)
})
