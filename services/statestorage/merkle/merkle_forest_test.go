package merkle

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func updateStringEntries(f *Forest, baseHash primitives.MerkleSha256, keyValues ...string) primitives.MerkleSha256 {
	if len(keyValues)%2 != 0 {
		panic("expected key value pairs")
	}
	diffs := make(MerkleDiffs)
	for i := 0; i < len(keyValues); i = i + 2 {
		diffs[keyValues[i]] = hash.CalcSha256([]byte(keyValues[i+1]))
	}

	currentRoot, _ := f.Update(baseHash, diffs)

	return currentRoot
}

func verifyProof(t *testing.T, f *Forest, root primitives.MerkleSha256, proof Proof, path string, value string, exists bool) {
	verified, err := f.Verify(root, proof, path, hash.CalcSha256([]byte(value)))
	require.NoError(t, err, "proof verification failed")
	require.Equal(t, exists, verified, "proof verification returned unexpected result")
}

func getProofRequireHeight(t *testing.T, f *Forest, root primitives.MerkleSha256, path string, expectedHeight int) Proof {
	proof, err := f.GetProof(root, path)
	require.NoError(t, err, "failed with error: %s", err)
	require.Equal(t, expectedHeight, len(proof), "unexpected proof length")
	return proof
}

func TestRootManagement(t *testing.T) {
	f, _ := NewForest()

	require.Len(t, f.roots, 1, "new forest should have 1 root")
	emptyNode := createEmptyNode()
	foundRoot := f.findRoot(emptyNode.hash)
	require.Equal(t, emptyNode, foundRoot, "proof verification returned unexpected result")

	node1 := createNode("hi", hash.CalcSha256([]byte("bye")), true)
	node1.hash = node1.serialize().hash()
	node2 := createNode("bye", hash.CalcSha256([]byte("d")), true)
	node2.hash = node2.serialize().hash()

	f.appendRoot(node1)
	f.appendRoot(node2)
	require.Len(t, f.roots, 3, "mismatch length")

	node1hash := createNode("hi", hash.CalcSha256([]byte("bye")), true).serialize().hash()
	foundRoot = f.findRoot(node1hash)
	require.Equal(t, node1, foundRoot, "should be same node")

	node2hash := createNode("bye", hash.CalcSha256([]byte("d")), true).serialize().hash()
	f.Forget(node2hash)
	require.Len(t, f.roots, 2, "mismatch length after forget")
	foundRoot = f.findRoot(node1.hash)
	require.Equal(t, node1, foundRoot, "should be same node")
	foundRoot = f.findRoot(node2.hash)
	require.Nil(t, foundRoot, "node should not exist")
}

func TestAddSingleEntryToEmptyTree(t *testing.T) {
	f, root := NewForest()
	root = updateStringEntries(f, root, "bar", "baz")

	getProofRequireHeight(t, f, root, "bar", 1)
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
	proof := getProofRequireHeight(t, f, root, key, 1)
	verifyProof(t, f, root, proof, key, "", true)
	verifyProof(t, f, root, proof, key, "non-zero", false)
}

func TestUpdateTrieFailsForMissingBaseNode(t *testing.T) {
	f, _ := NewForest()
	badroot := primitives.MerkleSha256(hash.CalcSha256([]byte("justanytext")))

	root := updateStringEntries(f, badroot, "first", "val")

	require.Nil(t, root, "did not receive an empty response when using a corrupt merkle root")
}

func TestProofValidationAfterBatchStateUpdate(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "bar1", "baz", "shared", "quux1")

	proof := getProofRequireHeight(t, f, root1, "bar1", 2)
	verifyProof(t, f, root1, proof, "bar1", "baz", true)
	proof = getProofRequireHeight(t, f, root1, "bar2", 2)
	verifyProof(t, f, root1, proof, "bar2", "qux", false)

	root2 := updateStringEntries(f, root1, "bar2", "qux", "shared", "quux2")
	require.NotEqual(t, root2, root1, "roots should be different")

	// retest that first insert proof still valid
	proof = getProofRequireHeight(t, f, root1, "bar1", 2)
	verifyProof(t, f, root1, proof, "bar1", "baz", true)
	proof = getProofRequireHeight(t, f, root1, "bar2", 2)
	verifyProof(t, f, root1, proof, "bar2", "qux", false)
	proof = getProofRequireHeight(t, f, root1, "shared", 2)
	verifyProof(t, f, root1, proof, "shared", "quux1", true)
	verifyProof(t, f, root1, proof, "shared", "quux2", false)

	// after second insert proofs
	proof = getProofRequireHeight(t, f, root2, "bar2", 3)
	verifyProof(t, f, root2, proof, "bar2", "qux", true)
	proof = getProofRequireHeight(t, f, root2, "bar1", 3)
	verifyProof(t, f, root2, proof, "bar1", "baz", true)
	proof = getProofRequireHeight(t, f, root2, "shared", 2)
	verifyProof(t, f, root2, proof, "shared", "quux2", true)
	verifyProof(t, f, root2, proof, "shared", "quux1", false)
}

func TestProofValidationForTwoRevisionsOfSameKey(t *testing.T) {
	f, root := NewForest()
	root1 := updateStringEntries(f, root, "bar1", "baz1")
	root2 := updateStringEntries(f, root1, "bar1", "baz2")

	proof := getProofRequireHeight(t, f, root1, "bar1", 1)
	verifyProof(t, f, root1, proof, "bar1", "baz1", true)

	proof = getProofRequireHeight(t, f, root2, "bar1", 1)
	verifyProof(t, f, root2, proof, "bar1", "baz2", true)
}

func TestExtendingLeafNodeWithNoBranchesOneAtATime(t *testing.T) {
	f, root := NewForest()
	root1 := updateStringEntries(f, root, "ba", "zoo")
	root2 := updateStringEntries(f, root1, "bar", "baz")
	root3 := updateStringEntries(f, root2, "baron", "Hello")

	getProofRequireHeight(t, f, root3, "baron", 3)
}

func TestExtendingLeafNodeWithNoBranches(t *testing.T) {
	f, root := NewForest()
	root = updateStringEntries(f, root, "ba", "zoo", "bar", "baz", "baron", "Hello")

	getProofRequireHeight(t, f, root, "baron", 3)
}

func TestExtendingLeafNodeWithNoBranchesInWrongOrder(t *testing.T) {
	f, root := NewForest()
	root = updateStringEntries(f, root, "baron", "Hello", "bar", "baz", "ba", "zoo")

	getProofRequireHeight(t, f, root, "baron", 3)
}

func TestExtendingKeyPathByOneChar(t *testing.T) {
	f, root := NewForest()
	root = updateStringEntries(f, root, "bar", "baz", "bar1", "qux")

	proof := getProofRequireHeight(t, f, root, "bar1", 2)
	verifyProof(t, f, root, proof, "bar1", "qux", true)
}

func TestExtendingKeyPathBySeveralChars(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "bar", "baz", "bar12", "qux", "bar123456789", "quux")

	proof := getProofRequireHeight(t, f, root1, "bar123456789", 3)
	verifyProof(t, f, root1, proof, "bar123456789", "quux", true)
}

func TestAddSiblingNode(t *testing.T) {
	f, root := NewForest()
	root1 := updateStringEntries(f, root, "bar", "baz", "bar1", "qux", "bar2", "quux")

	proof := getProofRequireHeight(t, f, root1, "bar2", 2)
	verifyProof(t, f, root1, proof, "bar2", "quux", true)
}

func TestAddPathToCauseBranchingAlongExistingPath(t *testing.T) {
	f, root := NewForest()
	root1 := updateStringEntries(f, root, "bar", "baz", "bar1", "qux", "bad", "quux")

	proof := getProofRequireHeight(t, f, root1, "bad", 2)
	verifyProof(t, f, root1, proof, "bad", "quux", true)
}

func TestReplaceExistingValueBelowDivergingPaths(t *testing.T) {
	f, root := NewForest()
	root1 := updateStringEntries(f, root, "bar", "baz", "bar1", "qux", "bad", "quux", "bar1", "zoo")

	proof := getProofRequireHeight(t, f, root1, "bar1", 3)
	verifyProof(t, f, root1, proof, "bar1", "zoo", true)
	verifyProof(t, f, root1, proof, "bar1", "qux", false)
}

func TestAddPathToCauseNewLeafAlongExistingPath(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "baron", "Hirsch", "bar", "Hello")

	proof := getProofRequireHeight(t, f, root1, "bar", 1)
	verifyProof(t, f, root1, proof, "bar", "Hello", true)

	proof = getProofRequireHeight(t, f, root1, "baron", 2)
	verifyProof(t, f, root1, proof, "baron", "Hirsch", true)
}

func TestRemoveValue_SingleExistingNode(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "aKey", "aValue")
	root2 := updateStringEntries(f, root1, "aKey", "")

	getProofRequireHeight(t, f, root, "aKey", 1)
	getProofRequireHeight(t, f, root1, "aKey", 1)
	getProofRequireHeight(t, f, root2, "aKey", 1)
	require.EqualValues(t, root, root2, "for identical states hash must be identical")
	require.NotEqual(t, root1, root2, "for different states hash must be different")
}

func TestRemoveValue_RemoveSingleChildLeaf(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "prefix", "1")
	root2 := updateStringEntries(f, root1, "prefixSuffix", "2")
	root3 := updateStringEntries(f, root2, "prefixSuffix", "")

	getProofRequireHeight(t, f, root1, "prefixSuffix", 1)
	getProofRequireHeight(t, f, root2, "prefixSuffix", 2)
	getProofRequireHeight(t, f, root3, "prefixSuffix", 1)
	require.EqualValues(t, root1, root3, "root hash should be identical")
}

func TestRemoveValue_ParentWithSingleChild(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "no", "1", "noam", "1", "no", "")

	p := getProofRequireHeight(t, f, root1, "noam", 1)
	require.EqualValues(t, "noam", p[0].path, "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_NonBranchingNonLeaf1(t *testing.T) {
	f, root := NewForest()

	fullTree := updateStringEntries(f, root, "a", "1", "and", "2", "android", "3")
	afterRemove := updateStringEntries(f, fullTree, "and", "")

	p1 := getProofRequireHeight(t, f, fullTree, "and", 2)
	p2 := getProofRequireHeight(t, f, afterRemove, "and", 2)

	getProofRequireHeight(t, f, fullTree, "android", 3)
	getProofRequireHeight(t, f, afterRemove, "android", 2)

	require.EqualValues(t, "d", p1[1].path, "full tree proof for and does not end with expected node path")
	require.EqualValues(t, "droid", p2[1].path, "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_NonBranchingNonLeaf2(t *testing.T) {
	f, root := NewForest()

	fullTree := updateStringEntries(f, root, "an", "1", "and", "2", "android", "3")
	afterRemove := updateStringEntries(f, fullTree, "and", "")

	p1 := getProofRequireHeight(t, f, fullTree, "and", 2)
	p2 := getProofRequireHeight(t, f, afterRemove, "and", 2)

	getProofRequireHeight(t, f, fullTree, "android", 3)
	getProofRequireHeight(t, f, afterRemove, "android", 2)

	require.EqualValues(t, "", p1[1].path, "full tree proof for and does not end with expected node path")
	require.EqualValues(t, "roid", p2[1].path, "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_BranchingNonLeaf_NodeStructureUnchanged(t *testing.T) {
	f, root := NewForest()

	fullTree := updateStringEntries(f, root, "and", "1", "andalusian", "1", "android", "1")
	afterRemove := updateStringEntries(f, fullTree, "and", "")

	p1 := getProofRequireHeight(t, f, afterRemove, "andalusian", 2)
	p2 := getProofRequireHeight(t, f, afterRemove, "android", 2)

	getProofRequireHeight(t, f, fullTree, "android", 2)
	getProofRequireHeight(t, f, afterRemove, "android", 2)

	require.EqualValues(t, "lusian", p1[1].path, "full tree proof for and does not end with expected node path")
	require.EqualValues(t, "oid", p2[1].path, "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_BranchingNonLeaf_CollapseBranch(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "no", "7", "noam", "8", "noan", "9")
	root2 := updateStringEntries(f, root1, "no", "")

	p0 := getProofRequireHeight(t, f, root1, "noam", 3)
	require.EqualValues(t, "no", p0[0].path, "unexpected proof structure")

	p := getProofRequireHeight(t, f, root2, "noam", 2)
	require.EqualValues(t, zeroValueHash, p[0].value, "unexpected proof structure")
	require.EqualValues(t, "noa", p[0].path, "unexpected proof structure")
}

func TestRemoveValue_OneOfTwoChildren(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "noa", "1", "noam", "1", "noan", "1")
	root2 := updateStringEntries(f, root1, "noan", "")

	p := getProofRequireHeight(t, f, root2, "noam", 2)
	getProofRequireHeight(t, f, root2, "noan", 1)
	require.EqualValues(t, "noa", p[0].path, "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_OneOfTwoChildrenCollapsingParent(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "noam", "8", "noan", "9", "noan", "")

	p := getProofRequireHeight(t, f, root1, "noam", 1)
	getProofRequireHeight(t, f, root1, "noan", 1)
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
	proof1, _ := f1.GetProof(root1, "bar1234")

	f2, initRoot2 := NewForest()
	root2 := updateStringEntries(f2, initRoot2, keyValue[var2[0]], keyValue[var2[0]+1], keyValue[var2[1]], keyValue[var2[1]+1],
		keyValue[var2[2]], keyValue[var2[2]+1], keyValue[var2[3]], keyValue[var2[3]+1], keyValue[var2[4]], keyValue[var2[4]+1])
	proof2, _ := f2.GetProof(root2, "bar1234")

	require.Equal(t, root1, root2, "unexpected different root hash")
	require.Equal(t, len(proof1), len(proof2), "unexpected different tree depth / proof lengths")
	require.Equal(t, proof1[3].hash(), proof2[3].hash(), "unexpected different leaf node hash")

	f3, initRoot3 := NewForest()
	root3 := updateStringEntries(f3, initRoot3, keyValue[var3[0]], keyValue[var3[0]+1], keyValue[var3[1]], keyValue[var3[1]+1],
		keyValue[var3[2]], keyValue[var3[2]+1], keyValue[var3[3]], keyValue[var3[3]+1], keyValue[var3[4]], keyValue[var3[4]+1])
	proof3, _ := f3.GetProof(root3, "bar1234")

	require.Equal(t, root2, root3, "unexpected different root hash")
	require.Equal(t, len(proof2), len(proof3), "unexpected different tree depth / proof lengths")
	require.Equal(t, proof2[3].hash(), proof3[3].hash(), "unexpected different leaf node hash")
}

func TestAddConvegingPathsWithExactValues(t *testing.T) {
	f, root := NewForest()
	root1 := updateStringEntries(f, root, "abdbda", "1", "abdcda", "1", "acdbda", "1", "acdcda", "1")
	root2 := updateStringEntries(f, root1, "abdcda", "2")

	proof1, _ := f.GetProof(root2, "abdbda")
	proof2, _ := f.GetProof(root2, "abdcda")
	proof3, _ := f.GetProof(root2, "acdbda")
	proof4, _ := f.GetProof(root2, "acdcda")

	verifyProof(t, f, root2, proof1, "abdbda", "1", true)
	verifyProof(t, f, root2, proof2, "abdcda", "2", true)
	verifyProof(t, f, root2, proof3, "acdbda", "1", true)
	verifyProof(t, f, root2, proof4, "acdcda", "1", true)
}

// =================
// Debug helpers
// =================

func (f *Forest) dump(t *testing.T) {
	t.Logf("---------------- TRIE BEGIN ------------------")
	for _, root := range f.roots {
		root.printNode(" Ω", 0, f, t)
	}
	t.Logf("---------------- TRIE END --------------------")
}

func (n *node) printNode(label string, depth int, trie *Forest, t *testing.T) {
	prefix := strings.Repeat(" ", depth)
	leafText := ""
	if n.hasValue() {
		leafText = fmt.Sprintf(": %v", n.value)
	}
	pathString := fmt.Sprintf("%s%s)%s", prefix, label, n.path)
	t.Logf("%s%s\n", pathString, leafText)
	for l, v := range n.branches {
		if v != nil {
			v.printNode(string([]byte{byte(l)}), depth+len(pathString)-1, trie, t)
		}
	}
}
