package merkle

import (
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)


var hextext ="0123456789abcdef"
func bytesToHexString(s []byte) string {
	hexText := ""
	for _, b := range s {
		hexText = hexText + string(hextext[b])
	}
	return hexText
}

func hexStringToBytes(s string) []byte {
	if (len(s) % 2) != 0 {
		panic("key value needs to be a hex representation of a byte array")
	}
	bytesKey := make([]byte, len(s)/2)
	hex.Decode(bytesKey, []byte(s))
	return bytesKey
}

func updateStringEntries(f *Forest, baseHash primitives.MerkleSha256, keyValues ...string) primitives.MerkleSha256 {
	if len(keyValues)%2 != 0 {
		panic("expected key value pairs")
	}
	diffs := make(MerkleDiffs, len(keyValues)/2)
	for i := 0; i < len(keyValues); i = i + 2 {
		diffs[i/2] = &MerkleDiff{Key: hexStringToBytes(keyValues[i]), Value : hash.CalcSha256([]byte(keyValues[i+1]))}
	}

	currentRoot, _ := f.Update(baseHash, diffs)

	return currentRoot
}

func verifyProof(t *testing.T, f *Forest, root primitives.MerkleSha256, proof Proof, path string, value string, exists bool) {
	verified, err := f.Verify(root, proof, hexStringToBytes(path), hash.CalcSha256([]byte(value)))
	require.NoError(t, err, "proof verification failed")
	require.Equal(t, exists, verified, "proof verification returned unexpected result")
}

func getProofRequireHeight(t *testing.T, f *Forest, root primitives.MerkleSha256, path string, expectedHeight int) Proof {
	proof, err := f.GetProof(root, hexStringToBytes(path))
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

	node1 := createNode([]byte("abcd"), hash.CalcSha256([]byte("bye")), true)
	node1.hash = node1.serialize().hash()
	node2 := createNode([]byte("1234"), hash.CalcSha256([]byte("d")), true)
	node2.hash = node2.serialize().hash()

	f.appendRoot(node1)
	f.appendRoot(node2)
	require.Len(t, f.roots, 3, "mismatch length")

	node1hash := createNode([]byte("abcd"), hash.CalcSha256([]byte("bye")), true).serialize().hash()
	foundRoot = f.findRoot(node1hash)
	require.Equal(t, node1, foundRoot, "should be same node")
}

func TestRootForgetWhenMultipleSameRootsAndKeepOrder(t *testing.T) {
	f, _ := NewForest()

	node1 := createNode([]byte("abcd"), hash.CalcSha256([]byte("bye")), true)
	node1.hash = node1.serialize().hash()
	node2 := createNode([]byte("1234"), hash.CalcSha256([]byte("d")), true)
	node2.hash = node2.serialize().hash()

	f.appendRoot(node1)
	f.appendRoot(node2)
	f.appendRoot(node1)
	f.appendRoot(node2)
	require.Len(t, f.roots, 5, "mismatch length")

	f.Forget(node2.hash)
	require.Len(t, f.roots, 4, "mismatch length after forget")
	require.Equal(t, node1.hash, f.roots[1].hash, "should be same node1 (1)")
	require.Equal(t, node1.hash, f.roots[2].hash, "should be same node1 (2)")
	require.Equal(t, node2.hash, f.roots[3].hash, "should be same node2")
}

func TestAddSingleEntryToEmptyTree(t *testing.T) {
	f, root := NewForest()
	root = updateStringEntries(f, root, "badd", "baz")

	getProofRequireHeight(t, f, root, "badd", 1)
}

func TestRootChangeAfterStateChange(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "abcdef", "val")
	root2 := updateStringEntries(f, root1, "abcdef", "val1")

	require.NotEqual(t, root1, root2, "root hash did not change after state change")
}

func TestRevertingStateChangeRevertsMerkleRoot(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "abcdef", "val")
	root2 := updateStringEntries(f, root1, "abcdef", "val1")
	root3 := updateStringEntries(f, root2, "abcdef", "val")

	require.Equal(t, root1, root3, "root hash did not revert back after resetting state")
}

func TestValidProofForMissingKey(t *testing.T) {
	f, root := NewForest()
	key := "deaddead"
	proof := getProofRequireHeight(t, f, root, key, 1)
	verifyProof(t, f, root, proof, key, "", true)
	verifyProof(t, f, root, proof, key, "non-zero", false)
}

func TestUpdateTrieFailsForMissingBaseNode(t *testing.T) {
	f, _ := NewForest()
	badroot := primitives.MerkleSha256(hash.CalcSha256([]byte("deaddead")))

	root := updateStringEntries(f, badroot, "abcdef", "val")

	require.Nil(t, root, "did not receive an empty response when using a corrupt merkle root")
}

func TestProofValidationForTwoRevisionsOfSameKey(t *testing.T) {
	f, root := NewForest()
	root1 := updateStringEntries(f, root, "abc1", "baz1")
	root2 := updateStringEntries(f, root1, "abc1", "baz2")

	proof := getProofRequireHeight(t, f, root1, "abc1", 1)
	verifyProof(t, f, root1, proof, "abc1", "baz1", true)

	proof = getProofRequireHeight(t, f, root2, "abc1", 1)
	verifyProof(t, f, root2, proof, "abc1", "baz2", true)
}

func TestExtendingLeafNode(t *testing.T) {
	// one at a time
	f1, root1 := NewForest()
	root11 := updateStringEntries(f1, root1, "ab", "zoo")
	root12 := updateStringEntries(f1, root11, "abc1", "baz")
	root13 := updateStringEntries(f1, root12, "abc123", "Hello")

	// all together
	f2, root2 := NewForest()
	root23 := updateStringEntries(f2, root2, "ab", "zoo", "abc1", "baz", "abc123", "Hello")

	// all together diff orer
	f3, root3 := NewForest()
	root33 := updateStringEntries(f3, root3, "abc123", "Hello", "abc1", "baz", "ab", "zoo")

	require.Equal(t, root13, root23, "should be same root")
	require.Equal(t, root13, root33, "should be same root")

	proof1 := getProofRequireHeight(t, f1, root13, "abc123", 3)
	proof3 := getProofRequireHeight(t, f3, root33, "abc123", 3)
	require.EqualValues(t, proof1, proof3, "proofs should be same")
}

func TestExtendingKeyPathBySeveralChars(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "ab", "baz", "ab12", "qux", "ab126789", "quux")

	proof := getProofRequireHeight(t, f, root1, "ab126789", 3)
	verifyProof(t, f, root1, proof, "ab126789", "quux", true)
}

func TestProofValidationAfterUpdateWithBothReplaceAndNewValue(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "abc1", "baz", "ffeedd", "quux1")

	proof := getProofRequireHeight(t, f, root1, "abc1", 2)
	verifyProof(t, f, root1, proof, "abc1", "baz", true)
	proof = getProofRequireHeight(t, f, root1, "abc2", 2)
	verifyProof(t, f, root1, proof, "abc2", "qux", false)

	root2 := updateStringEntries(f, root1, "abc2", "qux", "ffeedd", "quux2")
	require.NotEqual(t, root2, root1, "roots should be different")

	// retest that first insert proof still valid
	proof = getProofRequireHeight(t, f, root1, "abc1", 2)
	verifyProof(t, f, root1, proof, "abc1", "baz", true)
	proof = getProofRequireHeight(t, f, root1, "abc2", 2)
	verifyProof(t, f, root1, proof, "abc2", "qux", false)
	proof = getProofRequireHeight(t, f, root1, "ffeedd", 2)
	verifyProof(t, f, root1, proof, "ffeedd", "quux1", true)
	verifyProof(t, f, root1, proof, "ffeedd", "quux2", false)

	// after second insert proofs
	proof = getProofRequireHeight(t, f, root2, "abc2", 3)
	verifyProof(t, f, root2, proof, "abc2", "qux", true)
	proof = getProofRequireHeight(t, f, root2, "abc1", 3)
	verifyProof(t, f, root2, proof, "abc1", "baz", true)
	proof = getProofRequireHeight(t, f, root2, "ffeedd", 2)
	verifyProof(t, f, root2, proof, "ffeedd", "quux2", true)
	verifyProof(t, f, root2, proof, "ffeedd", "quux1", false)
}

func TestAddSiblingNode(t *testing.T) {
	f, root := NewForest()
	root1 := updateStringEntries(f, root, "ab", "baz", "abc1", "qux", "abd1", "quux")

	proof := getProofRequireHeight(t, f, root1, "abd1", 2)
	verifyProof(t, f, root1, proof, "abd1", "quux", true)
}

func TestAddPathToCauseBranchingAlongExistingPath(t *testing.T) {
	f, root := NewForest()
	root1 := updateStringEntries(f, root, "ab", "baz", "abc1", "qux", "abf0", "quux")

	proof := getProofRequireHeight(t, f, root1, "abf0", 2)
	verifyProof(t, f, root1, proof, "abf0", "quux", true)
}

func TestReplaceExistingValueBelowDivergingPaths(t *testing.T) {
	f, root := NewForest()
	root1 := updateStringEntries(f, root, "ab", "baz", "abc1", "qux", "abc2", "bar", "abf0", "quux")
	root2 := updateStringEntries(f, root1, "abc1", "zoo")

	proof := getProofRequireHeight(t, f, root2, "abc1", 3)
	verifyProof(t, f, root2, proof, "abc1", "zoo", true)
	verifyProof(t, f, root2, proof, "abc1", "qux", false)
}

func TestAddPathToCauseNewParent(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "abc123", "Hirsch", "abc1", "Hello")

	proof := getProofRequireHeight(t, f, root1, "abc1", 1)
	verifyProof(t, f, root1, proof, "abc1", "Hello", true)

	proof = getProofRequireHeight(t, f, root1, "abc123", 2)
	verifyProof(t, f, root1, proof, "abc123", "Hirsch", true)
}

func TestRemoveValue_SingleExistingNode(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "ab", "aValue")
	root2 := updateStringEntries(f, root1, "ab", "")

	getProofRequireHeight(t, f, root, "ab", 1)
	getProofRequireHeight(t, f, root1, "ab", 1)
	getProofRequireHeight(t, f, root2, "ab", 1)
	require.EqualValues(t, root, root2, "for identical states hash must be identical")
	require.NotEqual(t, root1, root2, "for different states hash must be different")
}

func TestRemoveValue_RemoveSingleChildLeaf(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "abcd", "1")
	root2 := updateStringEntries(f, root1, "abcdef", "2")
	root3 := updateStringEntries(f, root2, "abcdef", "")

	getProofRequireHeight(t, f, root1, "abcdef", 1)
	getProofRequireHeight(t, f, root2, "abcdef", 2)
	getProofRequireHeight(t, f, root3, "abcdef", 1)
	require.EqualValues(t, root1, root3, "root hash should be identical")
}

func TestRemoveValue_ParentWithSingleChild(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "abcd", "1", "abcdef", "1")
	root2 := updateStringEntries(f, root1, "abcd", "")

	p := getProofRequireHeight(t, f, root2, "abcdef", 1)
	require.EqualValues(t, "abcdef", bytesToHexString(p[0].path), "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_NonBranchingNonLeaf1(t *testing.T) {
	f, root := NewForest()

	fullTree := updateStringEntries(f, root, "ab", "1", "abcd", "2", "abcdef", "3")
	afterRemove := updateStringEntries(f, fullTree, "abcd", "")

	p1 := getProofRequireHeight(t, f, fullTree, "abcd", 2)
	p2 := getProofRequireHeight(t, f, afterRemove, "abcd", 2)

	getProofRequireHeight(t, f, fullTree, "abcdef", 3)
	getProofRequireHeight(t, f, afterRemove, "abcdef", 2)

	require.EqualValues(t, "d", bytesToHexString(p1[1].path), "full tree proof for and does not end with expected node path")
	require.EqualValues(t, "def", bytesToHexString(p2[1].path), "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_BranchingNonLeaf_NodeStructureUnchanged(t *testing.T) {
	f, root := NewForest()

	fullTree := updateStringEntries(f, root, "ab", "1", "ab1234", "1", "abcdef", "1")
	afterRemove := updateStringEntries(f, fullTree, "ab", "")

	p1 := getProofRequireHeight(t, f, afterRemove, "ab1234", 2)
	p2 := getProofRequireHeight(t, f, afterRemove, "abcdef", 2)

	getProofRequireHeight(t, f, fullTree, "abcdef", 2)
	getProofRequireHeight(t, f, afterRemove, "abcdef", 2)

	require.EqualValues(t, "234", bytesToHexString(p1[1].path), "full tree proof for and does not end with expected node path")
	require.EqualValues(t, "def", bytesToHexString(p2[1].path), "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_BranchingNonLeaf_CollapseRoot(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "ab", "7", "abcd", "8", "abce", "9")
	root2 := updateStringEntries(f, root1, "ab", "")

	p0 := getProofRequireHeight(t, f, root1, "abcd", 3)
	require.EqualValues(t, "ab", bytesToHexString(p0[0].path), "unexpected proof structure")

	p := getProofRequireHeight(t, f, root2, "abcd", 2)
	require.EqualValues(t, zeroValueHash, p[0].value, "unexpected proof structure")
	require.EqualValues(t, "abc", bytesToHexString(p[0].path), "unexpected proof structure")
}

func TestRemoveValue_OneOfTwoChildren(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "ab", "1", "abcdef", "1", "ab1234", "1")
	root2 := updateStringEntries(f, root1, "ab1234", "")

	p := getProofRequireHeight(t, f, root2, "abcdef", 2)
	getProofRequireHeight(t, f, root2, "ab1234", 1)
	require.EqualValues(t, "ab", bytesToHexString(p[0].path), "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_OneOfTwoChildrenCollapsingParent(t *testing.T) {
	f, root := NewForest()

	root1 := updateStringEntries(f, root, "abcd", "8", "abc4", "9")
	root2 := updateStringEntries(f, root1, "abc4", "")

	p := getProofRequireHeight(t, f, root2, "abcd", 1)
	getProofRequireHeight(t, f, root2, "abc4", 1)
	require.EqualValues(t, "abcd", bytesToHexString(p[0].path), "unexpected proof structure")
}

func TestRemoveValue_MissingKey(t *testing.T) {
	f, root := NewForest()

	baseHash := updateStringEntries(f, root, "abc1ab", "1", "abc1ba", "1", "abc1abccdd", "1", "abc1abccee", "1")
	hash1 := updateStringEntries(f, baseHash, "abc2aa123456", "")
	hash2 := updateStringEntries(f, hash1, "abc1", "")
	hash3 := updateStringEntries(f, hash2, "ab", "")
	hash4 := updateStringEntries(f, hash3, "abc1abcc", "")

	require.EqualValues(t, baseHash, hash1, "tree changed after removing missing key")
	require.EqualValues(t, baseHash, hash2, "tree changed after removing missing key")
	require.EqualValues(t, baseHash, hash3, "tree changed after removing missing key")
	require.EqualValues(t, baseHash, hash4, "tree changed after removing missing key")
}

func TestOrderOfAdditionsDoesNotMatter(t *testing.T) {
	keyValue := []string{"abcd", "baz", "abc123", "qux", "abc12345", "quux", "aadd", "foo", "1234", "hello"}
	var1 := []int{2, 6, 0, 8, 4}
	var2 := []int{8, 4, 0, 2, 6}
	var3 := []int{8, 6, 4, 2, 0}

	f1, initRoot1 := NewForest()
	root1 := updateStringEntries(f1, initRoot1, keyValue[var1[0]], keyValue[var1[0]+1], keyValue[var1[1]], keyValue[var1[1]+1],
		keyValue[var1[2]], keyValue[var1[2]+1], keyValue[var1[3]], keyValue[var1[3]+1], keyValue[var1[4]], keyValue[var1[4]+1])
	proof1, _ := f1.GetProof(root1, hexStringToBytes("abc12345"))

	f2, initRoot2 := NewForest()
	root2 := updateStringEntries(f2, initRoot2, keyValue[var2[0]], keyValue[var2[0]+1], keyValue[var2[1]], keyValue[var2[1]+1],
		keyValue[var2[2]], keyValue[var2[2]+1], keyValue[var2[3]], keyValue[var2[3]+1], keyValue[var2[4]], keyValue[var2[4]+1])
	proof2, _ := f2.GetProof(root2, hexStringToBytes("abc12345"))

	require.Equal(t, root1, root2, "unexpected different root hash")
	require.Equal(t, len(proof1), len(proof2), "unexpected different tree depth / proof lengths")
	require.Equal(t, proof1[3].hash(), proof2[3].hash(), "unexpected different leaf node hash")

	f3, initRoot3 := NewForest()
	root3 := updateStringEntries(f3, initRoot3, keyValue[var3[0]], keyValue[var3[0]+1], keyValue[var3[1]], keyValue[var3[1]+1],
		keyValue[var3[2]], keyValue[var3[2]+1], keyValue[var3[3]], keyValue[var3[3]+1], keyValue[var3[4]], keyValue[var3[4]+1])
	proof3, _ := f3.GetProof(root3, hexStringToBytes("abc12345"))

	require.Equal(t, root2, root3, "unexpected different root hash")
	require.Equal(t, len(proof2), len(proof3), "unexpected different tree depth / proof lengths")
	require.Equal(t, proof2[3].hash(), proof3[3].hash(), "unexpected different leaf node hash")
}

func TestAddConvegingPathsWithExactValues(t *testing.T) {
	f, root := NewForest()
	root1 := updateStringEntries(f, root, "abdbda", "1", "abdcda", "1", "acdbda", "1", "acdcda", "1")
	root2 := updateStringEntries(f, root1, "abdcda", "2")

	proof1, _ := f.GetProof(root2, hexStringToBytes("abdbda"))
	proof2, _ := f.GetProof(root2, hexStringToBytes("abdcda"))
	proof3, _ := f.GetProof(root2, hexStringToBytes("acdbda"))
	proof4, _ := f.GetProof(root2, hexStringToBytes("acdcda"))

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

var hexValues ="012345679abcdef"
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
			v.printNode(string(hexValues[l]), depth+len(pathString)-1, trie, t)
		}
	}
}
