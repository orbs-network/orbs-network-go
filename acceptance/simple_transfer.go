package acceptance

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/orbs-network/orbs-network-go/types"
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/blockstorage"
	"github.com/orbs-network/orbs-network-go/events"
)

var _ = Describe("a leader node", func() {

	It("commits transactions to all nodes", func() {
		inMemoryGossip := gossip.NewPausableGossip()
		bp1 := blockstorage.NewInMemoryBlockPersistence("bp1")
		bp2 := blockstorage.NewInMemoryBlockPersistence("bp2")

		node1 := bootstrap.NewNode(inMemoryGossip, bp1, events.NewEvents(),true)
		node2 := bootstrap.NewNode(inMemoryGossip, bp2, events.NewEvents(),false)
		inMemoryGossip.RegisterAll([]gossip.Listener{node1, node2})

		node1.SendTransaction(&types.Transaction{Value: 17})
		node1.SendTransaction(&types.Transaction{Value: 97, Invalid: true})
		node1.SendTransaction(&types.Transaction{Value: 22})


		bp1.WaitForBlocks(2)
		Expect(node1.CallMethod()).To(Equal(39))
		bp2.WaitForBlocks(2)
		Expect(node2.CallMethod()).To(Equal(39))
	})
})

var _ = Describe("a non-leader node", func() {

	It("propagates transactions to leader but does not commit them itself", func() {
		inMemoryGossip := gossip.NewPausableGossip()
		bp1 := blockstorage.NewInMemoryBlockPersistence("bp1")
		bp2 := blockstorage.NewInMemoryBlockPersistence("bp2")
		node1 := bootstrap.NewNode(inMemoryGossip, bp1, events.NewEvents(),true)
		node2 := bootstrap.NewNode(inMemoryGossip, bp2, events.NewEvents(),false)
		inMemoryGossip.RegisterAll([]gossip.Listener{node1, node2})

		inMemoryGossip.PauseForwards()
		node2.SendTransaction(&types.Transaction{Value: 17})

		Expect(node1.CallMethod()).To(Equal(0))
		Expect(node2.CallMethod()).To(Equal(0))

		inMemoryGossip.ResumeForwards()

		bp1.WaitForBlocks(1)
		Expect(node1.CallMethod()).To(Equal(17))
		bp2.WaitForBlocks(1)
		Expect(node2.CallMethod()).To(Equal(17))
	})

})
