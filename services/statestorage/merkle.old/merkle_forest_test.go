package merkle

import (
	"encoding/base64"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

//TODO - serialization based on spec (oded)
//TODO - Radix 16 +/- parity

//TODO - avoid hashing values of less than 32 bytes ?? Other optimizations (see ethereum)?
//TODO - what hash functions should be used for values and what functions for node addresses?
//TODO - should we include full values or just hashes (compare Ethereum)

func updateStringEntries(f *Forest, baseHash primitives.MerkleSha256, keyValues ...string) primitives.MerkleSha256 {
	if len(keyValues)%2 != 0 {
		panic("expected key value pairs")
	}
	currentRoot := baseHash
	for i := 0; i < len(keyValues); i = i + 2 {
		currentRoot = f.updateSingleEntry(currentRoot, keyValues[i], hash.CalcSha256([]byte(keyValues[i+1])))
	}
	return currentRoot
}

func verifyProof(t *testing.T, f *Forest, root primitives.MerkleSha256, proof Proof, contract string, key string, value string, exists bool) {
	verified, err := f.Verify(root, proof, contract, key, value)
	require.NoError(t, err, "proof verification failed")
	require.Equal(t, exists, verified, "proof verification returned unexpected result")
}

func getProofRequireHeight(t *testing.T, f *Forest, root primitives.MerkleSha256, contract string, key string, expectedHeight int) Proof {
	proof, err := f.GetProof(root, contract, key)
	require.NoError(t, err, "failed with error: %s", err)
	require.Len(t, proof, expectedHeight, "unexpected proof length")
	return proof
}

func TestAddSingleEntryToEmptyTree(t *testing.T) {
	f, root := NewForest()
	root = updateStringEntries(f, root, "bar", "baz")

	getProofRequireHeight(t, f, root, "", "bar", 1)
}

func TestRootChangeAfterStateChange(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "first", "val")
	root2 := updateStringEntries(f, root1, "first", "val1")

	require.NotEqual(t, root1, root2, "root hash did not change after state change")
}

func TestRevertingStateChangeRevertsMerkleRoot(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "first", "val")
	root2 := updateStringEntries(f, root1, "first", "val1")
	root3 := updateStringEntries(f, root2, "first", "val")

	require.Equal(t, root1, root3, "root hash did not revert back after resetting state")
}

func TestValidProofForMissingKey(t *testing.T) {
	f, root := NewForest()
	key := "imNotHere"
	contract := "foo"
	proof := getProofRequireHeight(t, f, root, contract, key, 1)
	verifyProof(t, f, root, proof, contract, key, "", true)
	verifyProof(t, f, root, proof, contract, key, "non-zero", false)

}

func TestUpdateTrieFailsForMissingBaseNode(t *testing.T) {
	f, root := NewForest()

	root[3] = root[3] + 1
	csd := builders.ContractStateDiff().WithContractName("foo").WithStringRecord("bar1", "baz").Build()
	_, err := f.Update(root, []*protocol.ContractStateDiff{csd})
	require.Error(t, err, "did not receive an error when using a corrupt merkle.old root")
}

func TestProofValidationAfterBatchStateUpdate(t *testing.T) {
	f, root := NewForest()

	csd1 := builders.ContractStateDiff().WithContractName("foo").
		WithStringRecord("bar1", "baz").WithStringRecord("shared", "quux1").Build()
	iterator1 := csd1.StateDiffsIterator()
	bar1 := iterator1.NextStateDiffs()
	shared1 := iterator1.NextStateDiffs()

	csd2 := builders.ContractStateDiff().WithContractName("foo").
		WithStringRecord("bar2", "qux").WithStringRecord("shared", "quux2").Build()
	iterator2 := csd2.StateDiffsIterator()
	bar2 := iterator2.NextStateDiffs()
	shared2 := iterator2.NextStateDiffs()

	root1, _ := f.Update(root, []*protocol.ContractStateDiff{csd1})
	root2, _ := f.Update(root1, []*protocol.ContractStateDiff{csd2})

	proof := getProofRequireHeight(t, f, root1, "foo", bar1.StringKey(), 2)
	verifyProof(t, f, root1, proof, "foo", bar1.StringKey(), bar1.StringValue(), true)

	proof = getProofRequireHeight(t, f, root1, "foo", bar2.StringKey(), 2)
	verifyProof(t, f, root1, proof, "foo", bar2.StringKey(), bar2.StringValue(), false)

	proof = getProofRequireHeight(t, f, root2, "foo", bar2.StringKey(), 3)
	verifyProof(t, f, root2, proof, "foo", bar2.StringKey(), bar2.StringValue(), true)

	proof = getProofRequireHeight(t, f, root2, "foo", bar1.StringKey(), 3)
	verifyProof(t, f, root2, proof, "foo", bar1.StringKey(), bar1.StringValue(), true)

	proof = getProofRequireHeight(t, f, root1, "foo", shared1.StringKey(), 2)
	verifyProof(t, f, root1, proof, "foo", shared1.StringKey(), shared1.StringValue(), true)
	verifyProof(t, f, root1, proof, "foo", shared1.StringKey(), shared2.StringValue(), false)

	proof = getProofRequireHeight(t, f, root2, "foo", shared2.StringKey(), 2)
	verifyProof(t, f, root2, proof, "foo", shared2.StringKey(), shared2.StringValue(), true)
	verifyProof(t, f, root2, proof, "foo", shared2.StringKey(), shared1.StringValue(), false)
}

func TestProofValidationForTwoRevisionsOfSameKey(t *testing.T) {
	f, root := NewForest()
	root1 := updateStringEntries(f, root, "bar1", "baz1")
	root2 := updateStringEntries(f, root1, "bar1", "baz2")

	proof := getProofRequireHeight(t, f, root1, "", "bar1", 1)
	verifyProof(t, f, root1, proof, "", "bar1", "baz1", true)

	proof = getProofRequireHeight(t, f, root2, "", "bar1", 1)
	verifyProof(t, f, root2, proof, "", "bar1", "baz2", true)
}

func TestExtendingLeafNodeWithNoBranchesAndNoValue(t *testing.T) {
	f, root := NewForest()
	root = updateStringEntries(f, root, "ba", "zoo", "bar", "baz", "baron", "Hello")

	getProofRequireHeight(t, f, root, "", "baron", 3)
}

func TestExtendingKeyPathByOneChar(t *testing.T) {
	f, root := NewForest()
	root = updateStringEntries(f, root, "bar", "baz", "bar1", "qux")

	proof := getProofRequireHeight(t, f, root, "", "bar1", 2)
	verifyProof(t, f, root, proof, "", "bar1", "qux", true)
}

func TestExtendingKeyPathBySeveralChars(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "bar", "baz", "bar12", "qux", "bar123456789", "quux")

	proof := getProofRequireHeight(t, f, root1, "", "bar123456789", 3)
	verifyProof(t, f, root1, proof, "", "bar123456789", "quux", true)
}

func TestAddSiblingNode(t *testing.T) {
	f, root := NewForest()
	root1 := updateStringEntries(f, root, "bar", "baz", "bar1", "qux", "bar2", "quux")

	proof := getProofRequireHeight(t, f, root1, "", "bar2", 2)
	verifyProof(t, f, root1, proof, "", "bar2", "quux", true)
}

func TestAddPathToCauseBranchingAlongExistingPath(t *testing.T) {
	f, root := NewForest()
	root1 := updateStringEntries(f, root, "bar", "baz", "bar1", "qux", "bad", "quux")

	proof := getProofRequireHeight(t, f, root1, "", "bad", 2)
	verifyProof(t, f, root1, proof, "", "bad", "quux", true)
}

func TestReplaceExistingValueBelowDivergingPaths(t *testing.T) {
	f, root := NewForest()
	root1 := updateStringEntries(f, root, "bar", "baz", "bar1", "qux", "bad", "quux", "bar1", "zoo")

	proof := getProofRequireHeight(t, f, root1, "", "bar1", 3)
	verifyProof(t, f, root1, proof, "", "bar1", "zoo", true)
	verifyProof(t, f, root1, proof, "", "bar1", "qux", false)
}

func TestAddPathToCauseNewLeafAlongExistingPath(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "baron", "Hirsch", "bar", "Hello")

	proof := getProofRequireHeight(t, f, root1, "", "bar", 1)
	verifyProof(t, f, root1, proof, "", "bar", "Hello", true)

	proof = getProofRequireHeight(t, f, root1, "", "baron", 2)
	verifyProof(t, f, root1, proof, "", "baron", "Hirsch", true)
}

func TestRemoveValue_SingleExistingNode(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "aKey", "aValue")
	root2 := updateStringEntries(f, root, "aKey", "")

	getProofRequireHeight(t, f, root, "", "aKey", 1)
	getProofRequireHeight(t, f, root1, "", "aKey", 1)
	getProofRequireHeight(t, f, root2, "", "aKey", 1)
	require.EqualValues(t, root, root2, "for identical states hash must be identical")
	require.NotEqual(t, root1, root2, "for different states hash must be different")
}

func TestRemoveValue_RemoveSingleChildLeaf(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "prefix", "1")
	root2 := updateStringEntries(f, root1, "prefixSuffix", "2")
	root3 := updateStringEntries(f, root2, "prefixSuffix", "")

	getProofRequireHeight(t, f, root1, "", "prefixSuffix", 1)
	getProofRequireHeight(t, f, root2, "", "prefixSuffix", 2)
	getProofRequireHeight(t, f, root3, "", "prefixSuffix", 1)
	require.EqualValues(t, root1, root3, "root hash should be identical")
}

func TestRemoveValue_ParentWithSingleChild(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "no", "1", "noam", "1", "no", "")

	p := getProofRequireHeight(t, f, root1, "", "noam", 1)
	require.EqualValues(t, "noam", p[0].path, "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_NonBranchingNonLeaf1(t *testing.T) {
	f, root := NewForest()

	fullTree := updateStringEntries(f, root, "a", "1", "and", "2", "android", "3")
	afterRemove := updateStringEntries(f, fullTree, "and", "")

	p1 := getProofRequireHeight(t, f, fullTree, "", "and", 2)
	p2 := getProofRequireHeight(t, f, afterRemove, "", "and", 2)

	getProofRequireHeight(t, f, fullTree, "", "android", 3)
	getProofRequireHeight(t, f, afterRemove, "", "android", 2)

	require.EqualValues(t, "d", p1[1].path, "full tree proof for and does not end with expected node path")
	require.EqualValues(t, "droid", p2[1].path, "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_NonBranchingNonLeaf2(t *testing.T) {
	f, root := NewForest()

	fullTree := updateStringEntries(f, root, "an", "1", "and", "2", "android", "3")
	afterRemove := updateStringEntries(f, fullTree, "and", "")

	p1 := getProofRequireHeight(t, f, fullTree, "", "and", 2)
	p2 := getProofRequireHeight(t, f, afterRemove, "", "and", 2)

	getProofRequireHeight(t, f, fullTree, "", "android", 3)
	getProofRequireHeight(t, f, afterRemove, "", "android", 2)

	require.EqualValues(t, "", p1[1].path, "full tree proof for and does not end with expected node path")
	require.EqualValues(t, "roid", p2[1].path, "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_BranchingNonLeaf_NodeStructureUnchanged(t *testing.T) {
	f, root := NewForest()

	fullTree := updateStringEntries(f, root, "and", "1", "andalusian", "1", "android", "1")
	afterRemove := updateStringEntries(f, fullTree, "and", "")

	p1 := getProofRequireHeight(t, f, afterRemove, "", "andalusian", 2)
	p2 := getProofRequireHeight(t, f, afterRemove, "", "android", 2)

	getProofRequireHeight(t, f, fullTree, "", "android", 2)
	getProofRequireHeight(t, f, afterRemove, "", "android", 2)

	require.EqualValues(t, "lusian", p1[1].path, "full tree proof for and does not end with expected node path")
	require.EqualValues(t, "oid", p2[1].path, "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_BranchingNonLeaf_CollapseBranch(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "no", "7", "noam", "8", "noan", "9")
	root2 := updateStringEntries(f, root1, "no", "")

	p0 := getProofRequireHeight(t, f, root1, "", "noam", 3)
	require.EqualValues(t, "no", p0[0].path, "unexpected proof structure")

	p := getProofRequireHeight(t, f, root2, "", "noam", 2)
	require.EqualValues(t, false, p[0].hasValue(), "unexpected proof structure")
	require.EqualValues(t, "noa", p[0].path, "unexpected proof structure")
}

func TestRemoveValue_OneOfTwoChildren(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "noa", "1", "noam", "1", "noan", "1")
	root2 := updateStringEntries(f, root1, "noan", "")

	p := getProofRequireHeight(t, f, root2, "", "noam", 2)
	getProofRequireHeight(t, f, root2, "", "noan", 1)
	require.EqualValues(t, "noa", p[0].path, "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_OneOfTwoChildrenCollapsingParent(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "noam", "8", "noan", "9", "noan", "")

	p := getProofRequireHeight(t, f, root1, "", "noam", 1)
	getProofRequireHeight(t, f, root1, "", "noan", 1)
	require.EqualValues(t, "noam", p[0].path, "unexpected proof structure")
}

func TestRemoveValue_MissingKey(t *testing.T) {
	f, root := NewForest()

	baseHash := updateStringEntries(f, root, "noam", "1", "noan", "1", "noamon", "1", "noamiko", "1")
	hash1 := updateStringEntries(f, baseHash, "noamiko_andSomeSuffix", "")
	hash2 := updateStringEntries(f, hash1, "noa", "")
	hash3 := updateStringEntries(f, hash2, "n", "")
	hash4 := updateStringEntries(f, hash3, "noamo", "")

	require.EqualValues(t, baseHash, hash1, "tree changed after removing missing key")
	require.EqualValues(t, baseHash, hash2, "tree changed after removing missing key")
	require.EqualValues(t, baseHash, hash3, "tree changed after removing missing key")
	require.EqualValues(t, baseHash, hash4, "tree changed after removing missing key")
}

func TestOrderOfAdditionsDoesNotMatter(t *testing.T) {
	keyValue := []string{"bar", "baz", "bar123", "qux", "bar1234", "quux", "bad", "foo", "bank", "hello"}
	var1 := []int{2, 6, 0, 8, 4}
	var2 := []int{8, 4, 0, 2, 6}
	var3 := []int{8, 6, 4, 2, 0}

	f1, initRoot1 := NewForest()
	root1 := updateStringEntries(f1, initRoot1, keyValue[var1[0]], keyValue[var1[0]+1], keyValue[var1[1]], keyValue[var1[1]+1],
		keyValue[var1[2]], keyValue[var1[2]+1], keyValue[var1[3]], keyValue[var1[3]+1], keyValue[var1[4]], keyValue[var1[4]+1])
	proof1, _ := f1.GetProof(root1, "", "bar1234")

	f2, initRoot2 := NewForest()
	root2 := updateStringEntries(f2, initRoot2, keyValue[var2[0]], keyValue[var2[0]+1], keyValue[var2[1]], keyValue[var2[1]+1],
		keyValue[var2[2]], keyValue[var2[2]+1], keyValue[var2[3]], keyValue[var2[3]+1], keyValue[var2[4]], keyValue[var2[4]+1])
	proof2, _ := f2.GetProof(root2, "", "bar1234")

	require.Equal(t, root1, root2, "unexpected different root hash")
	require.Equal(t, len(proof1), len(proof2), "unexpected different tree depth / proof lengths")
	require.Equal(t, proof1[3].hash(), proof2[3].hash(), "unexpected different leaf node hash")

	f3, initRoot3 := NewForest()
	root3 := updateStringEntries(f3, initRoot3, keyValue[var3[0]], keyValue[var3[0]+1], keyValue[var3[1]], keyValue[var3[1]+1],
		keyValue[var3[2]], keyValue[var3[2]+1], keyValue[var3[3]], keyValue[var3[3]+1], keyValue[var3[4]], keyValue[var3[4]+1])
	proof3, _ := f3.GetProof(root3, "", "bar1234")

	require.Equal(t, root2, root3, "unexpected different root hash")
	require.Equal(t, len(proof2), len(proof3), "unexpected different tree depth / proof lengths")
	require.Equal(t, proof2[3].hash(), proof3[3].hash(), "unexpected different leaf node hash")
}

func TestAddConvegingPathsWithExactValues(t *testing.T) {
	f, root := NewForest()
	root1 := updateStringEntries(f, root, "abdbda", "1", "abdcda", "1", "acdbda", "1", "acdcda", "1")
	root2 := updateStringEntries(f, root1, "abdcda", "2")

	proof1, _ := f.GetProof(root2, "", "abdbda")
	proof2, _ := f.GetProof(root2, "", "abdcda")
	proof3, _ := f.GetProof(root2, "", "acdbda")
	proof4, _ := f.GetProof(root2, "", "acdcda")

	proof3.dump()
	proof4.dump()

	verifyProof(t, f, root2, proof1, "", "abdbda", "1", true)
	verifyProof(t, f, root2, proof2, "", "abdcda", "2", true)
	verifyProof(t, f, root2, proof3, "", "acdbda", "1", true)
	verifyProof(t, f, root2, proof4, "", "acdcda", "1", true)
}



// =================
// Debug helpers
// =================

func (f *Forest) dump(t *testing.T) {
	t.Logf("---------------- TRIE BEGIN ------------------")
	childNodes := make(map[string]*Node, len(f.nodes))
	for _, n := range f.nodes {
		for _, childHash := range n.branches {
			if len(childHash) != 0 {
				childNodes[childHash.KeyForMap()] = f.nodes[childHash.KeyForMap()]
			}
		}
	}

	for nodeHash := range f.nodes {
		if _, isChild := childNodes[nodeHash]; !isChild {
			f.nodes[nodeHash].printNode(" Ω", 0, f, t)
		}
	}
	t.Logf("---------------- TRIE END --------------------")
}

func (n *Node) printNode(label string, depth int, trie *Forest, t *testing.T) {
	prefix := strings.Repeat(" ", depth)
	leafText := ""
	if n.hasValue() {
		leafText = fmt.Sprintf(": %v", n.value)
	}
	pathString := fmt.Sprintf("%s%s)%s", prefix, label, n.path)
	t.Logf("%s%s\n", pathString, leafText)
	for l, v := range n.branches {
		if len(v) != 0 {
			trie.nodes[v.KeyForMap()].printNode(string([]byte{byte(l)}), depth+len(pathString)-1, trie, t)
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

// TODO - this just checks there are no data integrity in our driver integrity
func (f *Forest) testForestIntegrity(t *testing.T) {
	for h, n := range f.nodes {
		require.Equal(t, n.hash().KeyForMap(), h, "node key is not true hash code")
	}
}
