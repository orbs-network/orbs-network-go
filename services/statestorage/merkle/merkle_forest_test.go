package merkle

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

var _ bool = Describe("Merkle Forest", func() {

	When("querying for merkle root of specific generation", func() {
		It("returns the merkle root", func() {

		})
	})

	When("querying for top generation", func() {
		It("returns the current top", func() {

		})
	})

	When("adding a single value", func() {
		When("testing with wrong merkle root", func() {
			It("fails", func() {

			})
		})
		When("testing with wrong merkle value", func() {
			It("fails", func() {

			})
		})
		When("testing with correct merkle root and value", func() {
			It("succeeds", func() {
				f := NewForest()

				r := builders.ContractStateDiff().WithContractName("foo").WithStringRecord("bar", "baz").Build()
				k := r.StateDiffsIterator().NextStateDiffs().StringKey()
				v := r.StateDiffsIterator().NextStateDiffs().StringValue()
				f.Update(1, []*protocol.ContractStateDiff{r})

				proof, err := f.GetProof(1, "foo", k)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(proof)).To(Equal(1))
				exists, err := f.Verify(1, proof, "foo", k, v)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeTrue())
			})
		})
	})

	When("querying a value", func() {
		When("there are no entries", func() {
			It("verifies the value is not there", func() {
			})
		})
	})

	When("adding an empty state diff", func() {
		It("merkle root does not change", func() {
		})
	})

	When("adding a non empty partial diff on top of existing state", func() {
		It("is still possible to get valid proofs for all entries in previous state generations relative to old root", func() {

		})
		It("is possible to get valid proofs for all entries in new state generation relative to new root", func() {
			f := NewForest()

			diffContract := builders.ContractStateDiff().WithContractName("foo")
			r1 := diffContract.WithStringRecord("bar1", "baz").Build()
			k1 := r1.StateDiffsIterator().NextStateDiffs().StringKey()
			v1 := r1.StateDiffsIterator().NextStateDiffs().StringValue()
			f.Update(1, []*protocol.ContractStateDiff{r1})

			diffContract = builders.ContractStateDiff().WithContractName("foo")
			r2 := diffContract.WithStringRecord("bar2", "qux").Build()
			k2 := r2.StateDiffsIterator().NextStateDiffs().StringKey()
			v2 := r2.StateDiffsIterator().NextStateDiffs().StringValue()
			f.Update(2, []*protocol.ContractStateDiff{r2})

			proof1, err1 := f.GetProof(2, "foo", k1)
			Expect(err1).ToNot(HaveOccurred())
			exists1, err1 := f.Verify(2, proof1, "foo", k1, v1)
			Expect(err1).ToNot(HaveOccurred())
			Expect(exists1).To(BeTrue())

			proof2, err2 := f.GetProof(2, "foo", k2)
			Expect(err2).ToNot(HaveOccurred())
			exists2, err1 := f.Verify(2, proof2, "foo", k2, v2)
			Expect(err2).ToNot(HaveOccurred())
			Expect(exists2).To(BeTrue())
		})
		It("is possible to get valid proofs for the exclusion of a missing entry", func() {
		})
	})

	When("Adding Single Node in every iteration", func() {
		When("updating an existing root which is a leaf", func() {
			It("becomes the new root", func() {
				f := NewForest()

				diffContract := builders.ContractStateDiff().WithContractName("foo")
				r1 := diffContract.WithStringRecord("bar1", "baz").Build()
				f.Update(1, []*protocol.ContractStateDiff{r1})

				diffContract = builders.ContractStateDiff().WithContractName("foo")
				r2 := diffContract.WithStringRecord("bar1", "qux").Build()
				k2 := r2.StateDiffsIterator().NextStateDiffs().StringKey()
				v2 := r2.StateDiffsIterator().NextStateDiffs().StringValue()
				f.Update(2, []*protocol.ContractStateDiff{r2})

				proof, err := f.GetProof(2, "foo", k2)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(proof)).To(Equal(1))
				exists, err := f.Verify(2, proof, "foo", k2, v2)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeTrue())
			})
		})
		When("extending a tree by a second key longer by 1 character", func() {
			It("becomes the new leaf", func() {
				f := NewForest()

				diffContract := builders.ContractStateDiff().WithContractName("foo")
				r1 := diffContract.WithStringRecord("bar", "baz").Build()
				f.Update(1, []*protocol.ContractStateDiff{r1})

				diffContract = builders.ContractStateDiff().WithContractName("foo")
				r2 := diffContract.WithStringRecord("bar1", "qux").Build()
				k2 := r2.StateDiffsIterator().NextStateDiffs().StringKey()
				v2 := r2.StateDiffsIterator().NextStateDiffs().StringValue()
				f.Update(2, []*protocol.ContractStateDiff{r2})

				proof, err := f.GetProof(2, "foo", k2)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(proof)).To(Equal(2))
				exists, err := f.Verify(2, proof, "foo", k2, v2)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeTrue())
			})
		})
		When("extending a two level tree by yet another key longer by 1 char", func() {
			It("becomes the new leaf", func() {
				f := NewForest()

				rootId := f.updateStringEntries("bar", "baz", "bar123", "qux", "bar1234", "quux")
				proof, err := f.GetProof(rootId, "", "bar1234")
				Expect(err).ToNot(HaveOccurred())
				Expect(len(proof)).To(Equal(3))
				exists, err := f.Verify(rootId, proof, "", "bar1234", "quux")
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeTrue())
			})
		})
		When("extending a two level tree by yet another key longer by 1 char", func() {
			It("becomes the new leaf", func() {
				f := NewForest()

				rootId := f.updateStringEntries("bar", "baz", "bar12", "qux", "bar123456789", "quux")
				proof, err := f.GetProof(rootId, "", "bar123456789")
				Expect(err).ToNot(HaveOccurred())
				Expect(len(proof)).To(Equal(3))
				exists, err := f.Verify(rootId, proof, "", "bar123456789", "quux")
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeTrue())
			})
		})
		When("extending a two level tree by yet another key which can use the same branching node", func() {
			It("becomes the new leaf", func() {
				f := NewForest()

				rootId := f.updateStringEntries("bar", "baz", "bar1", "qux", "bar2", "quux")
				proof, err := f.GetProof(rootId, "", "bar2")
				Expect(err).ToNot(HaveOccurred())
				Expect(len(proof)).To(Equal(2))
				exists, err := f.Verify(rootId, proof, "", "bar2", "quux")
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeTrue())
			})
		})
		When("adding new value that splits a current branch into two levels", func() {
			It("becomes the new root", func() {
				f := NewForest()

				rootId := f.updateStringEntries("bar", "baz", "bar1", "qux", "bad", "quux")
				proof, err := f.GetProof(rootId, "", "bad")
				Expect(err).ToNot(HaveOccurred())
				Expect(len(proof)).To(Equal(2))
				exists, err := f.Verify(rootId, proof, "", "bad", "quux")
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeTrue())
			})
		})
	})
})
