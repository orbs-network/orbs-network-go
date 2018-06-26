package acceptance

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-network-go/types"
)

var _ = Describe("a node", func() {

	It("commits transactions if it is the leader", func() {
		inMemoryGossip := gossip.NewGossip()
		node1 := bootstrap.NewNode(inMemoryGossip, true)
		node2 := bootstrap.NewNode(inMemoryGossip, false)
		inMemoryGossip.RegisterAll([]gossip.Listener{node1, node2})

		node1.SendTransaction(&types.Transaction{Value: 17})
		node1.SendTransaction(&types.Transaction{Value: 97, Invalid: true})
		node1.SendTransaction(&types.Transaction{Value: 22})

		Expect(node1.CallMethod()).To(Equal(39))
		Expect(node2.CallMethod()).To(Equal(39))
	})

	It("does not commit transactions if it is not the leader", func() {
		inMemoryGossip := gossip.NewGossip()
		node1 := bootstrap.NewNode(inMemoryGossip, true)
		node2 := bootstrap.NewNode(inMemoryGossip, false)
		inMemoryGossip.RegisterAll([]gossip.Listener{node1, node2})

		node2.SendTransaction(&types.Transaction{Value: 17})

		Expect(node1.CallMethod()).To(Equal(0))
		Expect(node2.CallMethod()).To(Equal(0))
	})

})
