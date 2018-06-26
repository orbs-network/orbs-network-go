package acceptance

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/gossip"
)

var _ = Describe("a node", func() {

	It("accumulates the state of multiple transactions", func() {
		inMemoryGossip := gossip.NewGossip()
		node1 := bootstrap.NewNode(inMemoryGossip)
		node2 := bootstrap.NewNode(inMemoryGossip)
		inMemoryGossip.RegisterAll([]gossip.Listener{node1, node2})

		node1.SendTransaction(17)
		node1.SendTransaction(22)

		storedValue := node2.CallMethod()
		Expect(storedValue).To(Equal(39))
	})

})
