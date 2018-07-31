package adapter

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
		)

var _ = Describe("Reading a Key", func() {
	When("not providing a contract name", func() {
		It("Returns an error", func() {
			d := NewInMemoryStatePersistence(&struct{}{})
			_, err := d.ReadState("")
			Expect(err).To(MatchError("missing contract name"))
		})
	})

	When("providing a non existing contract", func() {
		It("Returns an error", func() {
			d := NewInMemoryStatePersistence(&struct{}{})
			_, err := d.ReadState("foo")
			Expect(err).To(HaveOccurred())
		})
	})
})