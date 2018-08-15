package test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

var _ bool = Describe("Getting Merkle Root", func() {
	When("requesting state hash", func() {
		It("returns a hash code", func() {
			d := newStateStorageDriver(1)

			root, err := d.service.GetStateHash(&services.GetStateHashInput{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(root.StateRootHash)).ToNot(BeZero())
		})
	})
	When("requesting a future block within grace range", func() {
		It("times out and returns an error", func() {
			d := newStateStorageDriverWithGrace(1, 1, 1)

			output, err := d.service.GetStateHash(&services.GetStateHashInput{BlockHeight: 1})
			Expect(errors.Cause(err)).To(MatchError("timed out waiting for block at height 1"))
			Expect(output).To(BeNil())
		})
	})
	When("requesting a future block outside grace range", func() {
		It("returns an error immediately", func() {
			d := newStateStorageDriverWithGrace(1, 1, 1)

			output, err := d.service.GetStateHash(&services.GetStateHashInput{BlockHeight: 2})
			Expect(errors.Cause(err)).To(MatchError("requested future block outside of grace range"))
			Expect(output).To(BeNil())
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
