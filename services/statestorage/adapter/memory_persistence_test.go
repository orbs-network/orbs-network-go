package adapter

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reading a Key", func() {
	When("not providing a contract name", func() {
		It("Returns an error", func() {
			d := NewInMemoryStatePersistence()
			_, err := d.ReadState(0, "")
			Expect(err).To(MatchError("missing contract name"))
		})
	})

	When("providing a non existing contract", func() {
		It("Returns an error", func() {
			d := NewInMemoryStatePersistence()
			_, err := d.ReadState(0, "foo")
			Expect(err).To(HaveOccurred())
		})
	})
})
