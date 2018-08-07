package merkle

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"fmt"
	"strings"
	"encoding/base64"
)

var _ = Describe("Merkle Forest", func() {
	When("Verifying proofs", func() {
		When("querying for merkle root of specific generation", func() {
			It("returns the merkle root", func() {

			})
		})

		When("querying for top generation", func() {
			It("returns the current top", func() {

			})
		})

		When("querying a value", func() {
			When("there are no entries", func() {
				It("verifies the value is not there", func() {
					f := NewForest()
					proof, err := f.GetProof(0, "", "imNotHere")
					Expect(err).ToNot(HaveOccurred())
					Expect(len(proof)).To(Equal(1))

					verified, err := f.Verify(0, proof, "foo", "imNotHere", "")
					Expect(err).ToNot(HaveOccurred())
					Expect(verified).To(BeTrue())

					verified, err = f.Verify(0, proof, "foo", "imNotHere", "non Zero")
					Expect(err).ToNot(HaveOccurred())
					Expect(verified).To(BeFalse())
				})
			})
		})
	})

	When("Building Trees", func(){
		When("Adding Single Node in every iteration", func() {
			When("adding first node", func() {
				It("becomes the new root", func() {
					f := NewForest()

					rootId := f.updateStringEntries("bar", "baz")
					Expect(rootId).To(Equal(RootId(1)))
					proof, err1 := f.GetProof(rootId, "", "bar")
					Expect(err1).ToNot(HaveOccurred())
					Expect(len(proof)).To(Equal(1))
				})
			})
			When("updating forest with ContractStateDiff", func() {
				It("becomes the new root", func() {
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

					proof, err := f.GetProof(1, "foo", k1)
					Expect(err).ToNot(HaveOccurred())
					Expect(len(proof)).To(Equal(1))
					exists, err := f.Verify(1, proof, "foo", k1, v1)
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeTrue())

					proof, err = f.GetProof(1, "foo", k2)
					Expect(err).ToNot(HaveOccurred())
					Expect(len(proof)).To(Equal(1))
					exists, err = f.Verify(1, proof, "foo", k2, v2)
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeFalse())

					proof, err = f.GetProof(2, "foo", k2)
					Expect(err).ToNot(HaveOccurred())
					Expect(len(proof)).To(Equal(2))
					exists, err = f.Verify(2, proof, "foo", k2, v2)
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeTrue())

					proof, err = f.GetProof(2, "foo", k1)
					Expect(err).ToNot(HaveOccurred())
					Expect(len(proof)).To(Equal(2))
					exists, err = f.Verify(2, proof, "foo", k1, v1)
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeTrue())

				})
			})
			When("updating the root value twice", func() {
				It("for each generation we can obtain a proof for the correct version, both proofs a single nodes", func() {
					f := NewForest()

					rootId := f.updateStringEntries("bar1", "baz1", "bar1", "baz2")

					proof, err := f.GetProof(rootId-1, "", "bar1")
					Expect(err).ToNot(HaveOccurred())
					Expect(len(proof)).To(Equal(1))
					exists, err := f.Verify(rootId-1, proof, "", "bar1", "baz1")
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeTrue())

					proof, err = f.GetProof(rootId, "", "bar1")
					Expect(err).ToNot(HaveOccurred())
					Expect(len(proof)).To(Equal(1))
					exists, err = f.Verify(rootId, proof, "", "bar1", "baz2")
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeTrue())
				})
			})
			When("extending a leaf-node that has empty value (and no branches)", func() {
				It("empty node is replaced", func() {
					f := NewForest()

					rootId := f.updateStringEntries("ba", "zoo", "bar", "", "baron", "Hello")
					proof, err := f.GetProof(rootId, "", "baron")
					Expect(err).ToNot(HaveOccurred())
					Expect(len(proof)).To(Equal(2))
				})
			})
			When("extending a tree by a second key longer by 1 character", func() {
				It("becomes the new leaf", func() {
					f := NewForest()

					rootId := f.updateStringEntries("bar", "baz", "bar1", "qux")

					proof, err := f.GetProof(rootId, "", "bar1")
					Expect(err).ToNot(HaveOccurred())
					Expect(len(proof)).To(Equal(2))

					exists, err := f.Verify(2, proof, "", "bar1", "qux")
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
			When("extending a two level tree by yet another key longer by many chars", func() {
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
			When("adding new key-value that splits a upper level branch into two levels", func() {
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
			When("adding new key-value that replaces a leaf in a two levels tree", func() {
				It("replace the leaf value", func() {
					f := NewForest()

					rootId := f.updateStringEntries("bar", "baz", "bar1", "qux", "bad", "quux", "bar1", "zoo")
					proof, err := f.GetProof(rootId, "", "bar1")
					exists, err := f.Verify(rootId, proof, "", "bar1", "zoo")
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeTrue())
					exists, err = f.Verify(rootId, proof, "", "bar1", "qux")
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeFalse())
				})
			})
			When("adding new key-value that is prefix of existing root", func() {
				It("becomes the new root", func() {
					f := NewForest()

					rootId := f.updateStringEntries("baron", "Hirsch", "bar", "Hello")
					proof, err := f.GetProof(rootId, "", "bar")
					Expect(err).ToNot(HaveOccurred())
					Expect(len(proof)).To(Equal(1))
					exists, err := f.Verify(rootId, proof, "", "bar", "Hello")
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeTrue())

					proof, err = f.GetProof(rootId, "", "baron")
					Expect(err).ToNot(HaveOccurred())
					Expect(len(proof)).To(Equal(2))
					exists, err = f.Verify(rootId, proof, "", "baron", "Hirsch")
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeTrue())

				})
			})
			When("adding same nodes in different order", func() {
				It("get same tree", func() {
					keyValue := []string {"bar", "baz", "bar123", "qux", "bar1234", "quux", "bad", "foo", "bank", "hello"}
					var1 := []int {2,6,0,8,4}
					var2 := []int {8,4,0,2,6}
					var3 := []int {8,6,4,2,0}

					f1 := NewForest()
					rootId1 := f1.updateStringEntries(keyValue[var1[0]], keyValue[var1[0]+1],keyValue[var1[1]], keyValue[var1[1]+1],
						keyValue[var1[2]], keyValue[var1[2]+1],keyValue[var1[3]], keyValue[var1[3]+1],keyValue[var1[4]], keyValue[var1[4]+1])
					root1, _ := f1.GetRoot(rootId1)
					proof1, _ := f1.GetProof(rootId1, "", "bar1234")

					f2 := NewForest()
					rootId2 := f2.updateStringEntries(keyValue[var2[0]], keyValue[var2[0]+1],keyValue[var2[1]], keyValue[var2[1]+1],
						keyValue[var2[2]], keyValue[var2[2]+1],keyValue[var2[3]], keyValue[var2[3]+1],keyValue[var2[4]], keyValue[var2[4]+1])
					root2, _ := f2.GetRoot(rootId2)
					proof2, _ := f2.GetProof(rootId2, "", "bar1234")

					Expect(rootId2).To(Equal(rootId1))
					Expect(root2).To(Equal(root1))
					Expect(len(proof2)).To(Equal(len(proof1)))
					Expect(proof2[3].hash()).To(Equal(proof1[3].hash()))

					f3 := NewForest()
					rootId3 := f3.updateStringEntries(keyValue[var3[0]], keyValue[var3[0]+1],keyValue[var3[1]], keyValue[var3[1]+1],
						keyValue[var3[2]], keyValue[var3[2]+1],keyValue[var3[3]], keyValue[var3[3]+1],keyValue[var3[4]], keyValue[var3[4]+1])
					root3, _ := f3.GetRoot(rootId3)
					proof3, _ := f3.GetProof(rootId3, "", "bar1234")

					Expect(rootId2).To(Equal(rootId3))
					Expect(root2).To(Equal(root3))
					Expect(len(proof2)).To(Equal(len(proof3)))
					Expect(proof2[3].hash()).To(Equal(proof3[3].hash()))
				})
			})

		})

	})
})


//TODO - updateStringEntries should advance RootId only by one
//TODO - updateStringEntries - the bulk update version (optimize node access)
//TODO - Radix 16
//TODO - parity
//TODO - use hashes of contract names
//TODO - GetProof - accept an in memory list of cached nodes (to support bulk proof fetch).
//TODO - serialization based on spec
//TODO - split branch and node leafs (this can be limited to serializeation only)
//TODO - accept Node DB object
//TODO - garbage collection
//TODO - when setting zero values - compact - remove redundant nodes
//TODO - avoid hashing values of less than 32 bytes
//TODO - what hash functions should be used for values and what functions for node addresses?
//TODO - in case save key length is enforced - accept a key length in the forest constructor
//TODO - Prepare for GC (set values of older nodes to know when they were last valid)

//TODO - change verify and update types to []byte from strings


// Debug helpers
// TODO - we don't use any of these. but they are useful for debugging

func (f *Forest) dump() {
	fmt.Println("---------------- TRIE BEGIN ------------------")
	for i, h := range f.roots {
		label := " Ω"
		if int(i) == len(f.roots) - 1 {
			label = "*Ω"
		}
		f.nodes[h.KeyForMap()].printNode(label, 0, f)
	}
	fmt.Println("---------------- TRIE END --------------------")
}

func (n *Node) printNode(label string, depth int, trie *Forest) {
	prefix := strings.Repeat(" ", depth)
	leafText := ""
	if n.hasValue() {
		leafText = fmt.Sprintf(": %v", n.value)
	}
	pathString := fmt.Sprintf("%s%s)%s", prefix, label, n.path)
	fmt.Printf("%s%s\n", pathString, leafText)
	for l, v := range n.branches {
		if len(v) != 0{
			trie.nodes[v.KeyForMap()].printNode(string([]byte{byte(l)}), depth + len(pathString) - 1, trie)
		}
	}
}

func (p *Proof) dump() {
	fmt.Println("---------------- PROOF BEGIN ------------------")
	for _, n := range *p {
		hash2 := n.hash()
		fmt.Printf("%s\n%+v\n", base64.StdEncoding.EncodeToString(hash2[:]), n)
	}
	fmt.Println("---------------- PROOF END --------------------")
}

// TODO - this just checks there are no data integrity in our forest integrity
func (f *Forest) testForestIntegrity() {
	for h, n := range f.nodes {
		Expect(h).To(Equal(n.hash().KeyForMap()))
	}
	for _, root := range f.roots {
		Expect(f.nodes[root.KeyForMap()]).ToNot(BeEmpty())
	}
	Expect(f.topRoot).To(Equal(f.roots[RootId(len(f.roots))-1]))
}