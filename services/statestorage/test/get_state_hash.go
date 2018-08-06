package test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

var _ bool = Describe("Getting Merkle Root", func() {
	When("invoking method", func() {
		It("returns a hash code", func() {
			d := newStateStorageDriver(1)

			root, err := d.service.GetStateHash(&services.GetStateHashInput{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(root.StateRootHash)).ToNot(BeZero())
		})
	})

	When("changing state", func() {
		When("adding first key/value", func() {
			It("merkle root changes", func() {
				d := newStateStorageDriver(1)

				root1, err := d.service.GetStateHash(&services.GetStateHashInput{})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(root1.StateRootHash)).ToNot(BeZero())

				d.commitValuePairs("foo", "bar", "baz")

				root2, err1 := d.service.GetStateHash(&services.GetStateHashInput{primitives.BlockHeight(1)})
				Expect(err1).ToNot(HaveOccurred())
				Expect(len(root2.StateRootHash)).ToNot(BeZero())
				Expect(root1.StateRootHash).ToNot(BeEquivalentTo(root2.StateRootHash))
			})
		})
	})
})
