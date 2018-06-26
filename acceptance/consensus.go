package acceptance

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-network-go/types"
	"github.com/orbs-network/orbs-network-go/blockstorage"
)

var _ = Describe("a leader node", func() {

	It("must get validations by all nodes to commit a transaction", func() {
		inMemoryGossip := gossip.NewPausableGossip()
		bp1 := blockstorage.NewInMemoryBlockPersistence()
		bp2 := blockstorage.NewInMemoryBlockPersistence()
		node1 := bootstrap.NewNode(inMemoryGossip, bp1, true)
		node2 := bootstrap.NewNode(inMemoryGossip, bp2, false)
		inMemoryGossip.RegisterAll([]gossip.Listener{node1, node2})

		inMemoryGossip.PauseConsensus()
		node1.SendTransaction(&types.Transaction{Value: 17})
		node1.SendTransaction(&types.Transaction{Value: 97, Invalid: true})
		node1.SendTransaction(&types.Transaction{Value: 22})

		Expect(node1.CallMethod()).To(Equal(0))
		Expect(node2.CallMethod()).To(Equal(0))

		inMemoryGossip.ResumeConsensus()
		bp1.WaitForBlocks(3)
		Expect(node1.CallMethod()).To(Equal(39))
		bp2.WaitForBlocks(3)
		Expect(node2.CallMethod()).To(Equal(39))
	})
})
