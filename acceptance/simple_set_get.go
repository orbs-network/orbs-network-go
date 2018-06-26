package acceptance

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/orbs-network/orbs-network-go/bootstrap"
)

var _ = Describe("a node", func() {

	It("shows the value that was set when calling get", func() {
		node := bootstrap.NewNode()
		_, err := node.SendTransaction(50)
		Expect(err).ToNot(HaveOccurred())

		storedValue, err := node.CallMethod()
		Expect(err).ToNot(HaveOccurred())
		Expect(storedValue).To(Equal(50))
	})

})
