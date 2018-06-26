package acceptance

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/gossip"
)

var _ = Describe("a node", func() {

	It("shows the value that was set when calling get", func() {
		inMemoryGossip := gossip.NewGossip()
		node1 := bootstrap.NewNode(inMemoryGossip)
		node2 := bootstrap.NewNode(inMemoryGossip)
		inMemoryGossip.RegisterAll([]gossip.Listener{node1, node2})

		_, err := node1.SendTransaction(50)
		Expect(err).ToNot(HaveOccurred())

		storedValue, err := node2.CallMethod()
		Expect(err).ToNot(HaveOccurred())
		Expect(storedValue).To(Equal(50))
	})

})
