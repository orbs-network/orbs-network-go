package acceptance

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-network-go/types"
	"github.com/orbs-network/orbs-network-go/blockstorage"
	"github.com/orbs-network/orbs-network-go/events"
)

var _ = Describe("a leader node", func() {

	It("must get validations by all nodes to commit a transaction", func() {
		leaderEvents := events.NewEvents()
		inMemoryGossip := gossip.NewPausableGossip()
		leaderBp := blockstorage.NewInMemoryBlockPersistence("leaderBp")
		validatorBp := blockstorage.NewInMemoryBlockPersistence("validatorBp")

		leader := bootstrap.NewNode(inMemoryGossip, leaderBp, leaderEvents, true)
		validator := bootstrap.NewNode(inMemoryGossip, validatorBp, events.NewEvents(), false)
		inMemoryGossip.RegisterAll([]gossip.Listener{leader, validator})

		inMemoryGossip.FailConsensusRequests()
		leader.SendTransaction(&types.Transaction{Value: 17})

		leaderEvents.WaitForConsensusRounds(1)

		Expect(leader.CallMethod()).To(Equal(0))
		Expect(validator.CallMethod()).To(Equal(0))

		inMemoryGossip.PassConsensusRequests()
		leaderBp.WaitForBlocks(1)
		Expect(leader.CallMethod()).To(Equal(17))
		validatorBp.WaitForBlocks(1)
		Expect(validator.CallMethod()).To(Equal(17))
	})
})
