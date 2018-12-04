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

func TestRootManagement(t *testing.T) {
	f, _ := NewForest()

	require.Len(t, f.roots, 1, "new forest should have 1 root")
	emptyNode := createEmptyTrieNode()
	foundRoot := f.findRoot(emptyNode.hash)
	require.Equal(t, emptyNode, foundRoot, "proof verification returned unexpected result")

	node1 := createNode([]byte{0, 1, 0, 1}, hash.CalcSha256([]byte("bye")))
	node1.hash = hashTrieNode(node1)
	node2 := createNode([]byte{1, 1, 1, 1}, hash.CalcSha256([]byte("d")))
	node2.hash = hashTrieNode(node2)

	f.appendRoot(node1)
	f.appendRoot(node2)
	require.Len(t, f.roots, 3, "mismatch length")

	node1hash := hashTrieNode(createNode([]byte{0, 1, 0, 1}, hash.CalcSha256([]byte("bye"))))
	foundRoot = f.findRoot(node1hash)
	require.Equal(t, node1, foundRoot, "should be same node")
}

func TestRootForgetWhenMultipleSameRootsAndKeepOrder(t *testing.T) {
	f, _ := NewForest()

	node1 := createNode([]byte{0, 1, 0, 1}, hash.CalcSha256([]byte("bye")))
	node1.hash = hashTrieNode(node1)
	node2 := createNode([]byte{0, 1, 1, 1}, hash.CalcSha256([]byte("d")))
	node2.hash = hashTrieNode(node2)

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
	root = generalKeyUpdateEntries(f, root, "badd", "baz")

	generalKeyGetProofRequireHeight(t, f, root, "badd", 1)
}

func TestRootChangeAfterStateChange(t *testing.T) {
	f, root := NewForest()

	root1 := generalKeyUpdateEntries(f, root, "abcdef", "val")
	root2 := generalKeyUpdateEntries(f, root1, "abcdef", "val1")

	require.NotEqual(t, root1, root2, "root hash did not change after state change")
}

func TestRevertingStateChangeRevertsMerkleRoot(t *testing.T) {
	f, root := NewForest()

	root1 := generalKeyUpdateEntries(f, root, "abcdef", "val")
	root2 := generalKeyUpdateEntries(f, root1, "abcdef", "val1")
	root3 := generalKeyUpdateEntries(f, root2, "abcdef", "val")

	require.Equal(t, root1, root3, "root hash did not revert back after resetting state")
}

func TestValidProofForMissingKey(t *testing.T) {
	f, root := NewForest()
	key := "deaddead"
	proof := generalKeyGetProofRequireHeight(t, f, root, key, 1)
	generalKeyVerifyProof(t, f, root, proof, key, "", true)
	generalKeyVerifyProof(t, f, root, proof, key, "non-zero", false)
}

func TestUpdateTrieFailsForMissingBaseNode(t *testing.T) {
	f, _ := NewForest()
	badroot := hash.CalcSha256([]byte("deaddead"))

	root := generalKeyUpdateEntries(f, badroot, "abcdef", "val")

	require.Nil(t, root, "did not receive an empty response when using a corrupt merkle root")
}

func TestProofValidationForTwoRevisionsOfSameKey(t *testing.T) {
	f, root := NewForest()
	root1 := generalKeyUpdateEntries(f, root, "abc1", "baz1")
	root2 := generalKeyUpdateEntries(f, root1, "abc1", "baz2")

	proof := generalKeyGetProofRequireHeight(t, f, root1, "abc1", 1)
	generalKeyVerifyProof(t, f, root1, proof, "abc1", "baz1", true)

	proof = generalKeyGetProofRequireHeight(t, f, root2, "abc1", 1)
	generalKeyVerifyProof(t, f, root2, proof, "abc1", "baz2", true)
}

func TestExtendingLeafNode(t *testing.T) {
	// one at a time
	f1, root1 := NewForest()
	root11 := generalKeyUpdateEntries(f1, root1, "ab", "zoo")
	root12 := generalKeyUpdateEntries(f1, root11, "abc1", "baz")
	root13 := generalKeyUpdateEntries(f1, root12, "abc123", "Hello")

	// all together
	f2, root2 := NewForest()
	root23 := generalKeyUpdateEntries(f2, root2, "ab", "zoo", "abc1", "baz", "abc123", "Hello")

	// all together diff orer
	f3, root3 := NewForest()
	root33 := generalKeyUpdateEntries(f3, root3, "abc123", "Hello", "abc1", "baz", "ab", "zoo")

	require.Equal(t, root13, root23, "should be same root")
	require.Equal(t, root13, root33, "should be same root")

	proof1 := generalKeyGetProofRequireHeight(t, f1, root13, "abc123", 3)
	proof3 := generalKeyGetProofRequireHeight(t, f3, root33, "abc123", 3)
	require.EqualValues(t, proof1, proof3, "proofs should be same")
}

func TestExtendingKeyPathBySeveralChars(t *testing.T) {
	f, root := NewForest()

	root1 := generalKeyUpdateEntries(f, root, "ab", "baz", "ab12", "qux", "ab126789", "quux")

	proof := generalKeyGetProofRequireHeight(t, f, root1, "ab126789", 3)
	generalKeyVerifyProof(t, f, root1, proof, "ab126789", "quux", true)
}

func TestProofValidationAfterUpdateWithBothReplaceAndNewValue(t *testing.T) {
	f, root := NewForest()

	root1 := generalKeyUpdateEntries(f, root, "abc1", "baz", "1234", "quux1")

	proof := generalKeyGetProofRequireHeight(t, f, root1, "abc1", 2)
	generalKeyVerifyProof(t, f, root1, proof, "abc1", "baz", true)
	proof = generalKeyGetProofRequireHeight(t, f, root1, "abc2", 2)
	generalKeyVerifyProof(t, f, root1, proof, "abc2", "qux", false)

	root2 := generalKeyUpdateEntries(f, root1, "abc2", "qux", "1234", "quux2")
	require.NotEqual(t, root2, root1, "roots should be different")

	// retest that first insert proof still valid
	proof = generalKeyGetProofRequireHeight(t, f, root1, "abc1", 2)
	generalKeyVerifyProof(t, f, root1, proof, "abc1", "baz", true)
	proof = generalKeyGetProofRequireHeight(t, f, root1, "abc2", 2)
	generalKeyVerifyProof(t, f, root1, proof, "abc2", "qux", false)
	proof = generalKeyGetProofRequireHeight(t, f, root1, "1234", 2)
	generalKeyVerifyProof(t, f, root1, proof, "1234", "quux1", true)
	generalKeyVerifyProof(t, f, root1, proof, "1234", "quux2", false)

	// after second insert proofs
	proof = generalKeyGetProofRequireHeight(t, f, root2, "abc2", 3)
	generalKeyVerifyProof(t, f, root2, proof, "abc2", "qux", true)
	proof = generalKeyGetProofRequireHeight(t, f, root2, "abc1", 3)
	generalKeyVerifyProof(t, f, root2, proof, "abc1", "baz", true)
	proof = generalKeyGetProofRequireHeight(t, f, root2, "1234", 2)
	generalKeyVerifyProof(t, f, root2, proof, "1234", "quux2", true)
	generalKeyVerifyProof(t, f, root2, proof, "1234", "quux1", false)
}

func TestAddSiblingNode(t *testing.T) {
	f, root := NewForest()
	root1 := binaryKeyUpdateEntries(f, root, "00000000", "baz")
	root2 := binaryKeyUpdateEntries(f, root1, "0000000000000000", "qux", "0000000010000000", "quux")

	proof := binaryKeyGetProofRequireHeight(t, f, root2, "0000000010000000", 2)
	binaryKeyVerifyProof(t, f, root2, proof, "0000000010000000", "quux", true)
}

func TestAddPathToCauseBranchingAlongExistingPath(t *testing.T) {
	f, root := NewForest()
	root1 := binaryKeyUpdateEntries(f, root, "00000000", "baz", "0000000000000000", "qux")
	root2 := binaryKeyUpdateEntries(f, root1, "0000000010000000", "quux")

	proof := binaryKeyGetProofRequireHeight(t, f, root2, "0000000010000000", 2)
	binaryKeyVerifyProof(t, f, root2, proof, "0000000010000000", "quux", true)
}

func TestReplaceExistingValueBelowDivergingPaths(t *testing.T) {
	f, root := NewForest()
	root1 := binaryKeyUpdateEntries(f, root, "00000000", "baz", "0000000000000000", "qux", "0000000010000000", "bar", "0000000011000000", "quux")
	root2 := binaryKeyUpdateEntries(f, root1, "0000000000000000", "zoo")

	proof := binaryKeyGetProofRequireHeight(t, f, root2, "0000000000000000", 2)
	binaryKeyVerifyProof(t, f, root2, proof, "0000000000000000", "zoo", true)
	binaryKeyVerifyProof(t, f, root2, proof, "0000000000000000", "qux", false)
}

func TestAddPathToCauseNewParent(t *testing.T) {
	f, root := NewForest()

	root1 := generalKeyUpdateEntries(f, root, "abc123", "Hirsch", "abc1", "Hello")

	proof := generalKeyGetProofRequireHeight(t, f, root1, "abc1", 1)
	generalKeyVerifyProof(t, f, root1, proof, "abc1", "Hello", true)

	proof = generalKeyGetProofRequireHeight(t, f, root1, "abc123", 2)
	generalKeyVerifyProof(t, f, root1, proof, "abc123", "Hirsch", true)
}

func TestRemoveValue_SingleExistingNode(t *testing.T) {
	f, root := NewForest()

	root1 := generalKeyUpdateEntries(f, root, "ab", "aValue")
	root2 := generalKeyUpdateEntries(f, root1, "ab", "")

	generalKeyGetProofRequireHeight(t, f, root, "ab", 1)
	generalKeyGetProofRequireHeight(t, f, root1, "ab", 1)
	generalKeyGetProofRequireHeight(t, f, root2, "ab", 1)
	require.EqualValues(t, root, root2, "for identical states hash must be identical")
	require.NotEqual(t, root1, root2, "for different states hash must be different")
}

func TestRemoveValue_RemoveSingleChildLeaf(t *testing.T) {
	f, root := NewForest()

	root1 := generalKeyUpdateEntries(f, root, "abcd", "1")
	root2 := generalKeyUpdateEntries(f, root1, "abcdef", "2")
	root3 := generalKeyUpdateEntries(f, root2, "abcdef", "")

	generalKeyGetProofRequireHeight(t, f, root1, "abcdef", 1)
	generalKeyGetProofRequireHeight(t, f, root2, "abcdef", 2)
	generalKeyGetProofRequireHeight(t, f, root3, "abcdef", 1)
	require.EqualValues(t, root1, root3, "root hash should be identical")
}

func bytesToBinaryString(s []byte) string {
	text := ""
	for _, b := range s {
		if b == 0 {
			text = text + "0"
		} else {
			text = text + "1"
		}
	}
	return text
}

func TestRemoveValue_ParentWithSingleChild(t *testing.T) {
	f, root := NewForest()

	root1 := generalKeyUpdateEntries(f, root, "ab", "1", "abcd", "1")
	root2 := generalKeyUpdateEntries(f, root1, "ab", "")

	p := generalKeyGetProofRequireHeight(t, f, root2, "abcd", 1)
	require.EqualValues(t, "1010101111001101" /*"abcd"*/, bytesToBinaryString(p[0].path), "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_NonBranchingNonLeaf1(t *testing.T) {
	f, root := NewForest()

	fullTree := generalKeyUpdateEntries(f, root, "ab", "1", "abcd", "2", "abcdef", "3")
	afterRemove := generalKeyUpdateEntries(f, fullTree, "abcd", "")

	p1 := generalKeyGetProofRequireHeight(t, f, fullTree, "abcd", 2)
	p2 := generalKeyGetProofRequireHeight(t, f, afterRemove, "abcd", 2)

	generalKeyGetProofRequireHeight(t, f, fullTree, "abcdef", 3)
	generalKeyGetProofRequireHeight(t, f, afterRemove, "abcdef", 2)

	require.EqualValues(t, "1001101" /* "cd" without first bit*/, bytesToBinaryString(p1[1].path), "full tree proof for and does not end with expected node path")
	require.EqualValues(t, "100110111101111" /*"cdef" without first bit*/, bytesToBinaryString(p2[1].path), "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_BranchingNonLeaf_NodeStructureUnchanged(t *testing.T) {
	f, root := NewForest()

	fullTree := generalKeyUpdateEntries(f, root, "ab", "1", "ab12", "1", "abcd", "1")
	afterRemove := generalKeyUpdateEntries(f, fullTree, "ab", "")

	p1 := generalKeyGetProofRequireHeight(t, f, afterRemove, "ab12", 2)
	p2 := generalKeyGetProofRequireHeight(t, f, afterRemove, "abcd", 2)

	generalKeyGetProofRequireHeight(t, f, fullTree, "abcd", 2)
	generalKeyGetProofRequireHeight(t, f, afterRemove, "abcd", 2)

	require.EqualValues(t, "0010010" /* missing first bit of "12"*/, bytesToBinaryString(p1[1].path), "full tree proof for and does not end with expected node path")
	require.EqualValues(t, "1001101" /* missing first bit of "cd" */, bytesToBinaryString(p2[1].path), "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_BranchingNonLeaf_CollapseRoot(t *testing.T) {
	f, root := NewForest()

	root1 := generalKeyUpdateEntries(f, root, "ab", "7", "abcd", "8", "abce", "9")
	root2 := generalKeyUpdateEntries(f, root1, "ab", "")

	p0 := generalKeyGetProofRequireHeight(t, f, root1, "abcd", 3)
	require.EqualValues(t, "10101011" /*"ab"*/, bytesToBinaryString(p0[0].path), "unexpected proof structure")

	p := generalKeyGetProofRequireHeight(t, f, root2, "abcd", 2)
	require.EqualValues(t, zeroValueHash, p[0].value, "unexpected proof structure")

	require.EqualValues(t, "10101011110011" /*"abc"+first two bits of d/e*/, bytesToBinaryString(p[0].path), "unexpected proof structure")
}

func TestRemoveValue_OneOfTwoChildren(t *testing.T) {
	f, root := NewForest()

	root1 := generalKeyUpdateEntries(f, root, "ab", "1", "abcdef", "1", "ab1234", "1")
	root2 := generalKeyUpdateEntries(f, root1, "ab1234", "")

	p := generalKeyGetProofRequireHeight(t, f, root2, "abcdef", 2)
	generalKeyGetProofRequireHeight(t, f, root2, "ab1234", 1)
	require.EqualValues(t, "10101011" /*"ab"*/, bytesToBinaryString(p[0].path), "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_OneOfTwoChildrenCollapsingParent(t *testing.T) {
	f, root := NewForest()

	root1 := generalKeyUpdateEntries(f, root, "abcd", "8", "abc4", "9")
	root2 := generalKeyUpdateEntries(f, root1, "abc4", "")

	p := generalKeyGetProofRequireHeight(t, f, root2, "abcd", 1)
	generalKeyGetProofRequireHeight(t, f, root2, "abc4", 1)
	require.EqualValues(t, "1010101111001101" /*"abcd"*/, bytesToBinaryString(p[0].path), "unexpected proof structure")
}

func TestRemoveValue_MissingKey(t *testing.T) {
	f, root := NewForest()

	baseHash := generalKeyUpdateEntries(f, root, "abc1ab", "1", "abc1ba", "1", "abc1abccdd", "1", "abc1abccee", "1")
	hash1 := generalKeyUpdateEntries(f, baseHash, "abc2aa123456", "")
	hash2 := generalKeyUpdateEntries(f, hash1, "abc1", "")
	hash3 := generalKeyUpdateEntries(f, hash2, "ab", "")
	hash4 := generalKeyUpdateEntries(f, hash3, "abc1abcc", "")

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
	root1 := generalKeyUpdateEntries(f1, initRoot1, keyValue[var1[0]], keyValue[var1[0]+1], keyValue[var1[1]], keyValue[var1[1]+1],
		keyValue[var1[2]], keyValue[var1[2]+1], keyValue[var1[3]], keyValue[var1[3]+1], keyValue[var1[4]], keyValue[var1[4]+1])
	proof1 := generalKeyGetProof(t, f1, root1, "abc12345")
	//proof1, _ := f1.GetProof(root1, hexStringToBytes("abc12345"))

	f2, initRoot2 := NewForest()
	root2 := generalKeyUpdateEntries(f2, initRoot2, keyValue[var2[0]], keyValue[var2[0]+1], keyValue[var2[1]], keyValue[var2[1]+1],
		keyValue[var2[2]], keyValue[var2[2]+1], keyValue[var2[3]], keyValue[var2[3]+1], keyValue[var2[4]], keyValue[var2[4]+1])
	proof2 := generalKeyGetProof(t, f2, root2, "abc12345")

	require.Equal(t, root1, root2, "unexpected different root hash")
	require.Equal(t, len(proof1), len(proof2), "unexpected different tree depth / proof lengths")
	require.Equal(t, proof1[3].hash(), proof2[3].hash(), "unexpected different leaf node hash")

	f3, initRoot3 := NewForest()
	root3 := generalKeyUpdateEntries(f3, initRoot3, keyValue[var3[0]], keyValue[var3[0]+1], keyValue[var3[1]], keyValue[var3[1]+1],
		keyValue[var3[2]], keyValue[var3[2]+1], keyValue[var3[3]], keyValue[var3[3]+1], keyValue[var3[4]], keyValue[var3[4]+1])
	proof3 := generalKeyGetProof(t, f3, root3, "abc12345")

	require.Equal(t, root2, root3, "unexpected different root hash")
	require.Equal(t, len(proof2), len(proof3), "unexpected different tree depth / proof lengths")
	require.Equal(t, proof2[3].hash(), proof3[3].hash(), "unexpected different leaf node hash")
}

func TestAddConvegingPathsWithExactValues(t *testing.T) {
	f, root := NewForest()
	root1 := generalKeyUpdateEntries(f, root, "abdbda", "1", "abdcda", "1", "acdbda", "1", "acdcda", "1")
	root2 := generalKeyUpdateEntries(f, root1, "abdcda", "2")

	proof1 := generalKeyGetProof(t, f, root2, "abdbda")
	proof2 := generalKeyGetProof(t, f, root2, "abdcda")
	proof3 := generalKeyGetProof(t, f, root2, "acdbda")
	proof4 := generalKeyGetProof(t, f, root2, "acdcda")

	generalKeyVerifyProof(t, f, root2, proof1, "abdbda", "1", true)
	generalKeyVerifyProof(t, f, root2, proof2, "abdcda", "2", true)
	generalKeyVerifyProof(t, f, root2, proof3, "acdbda", "1", true)
	generalKeyVerifyProof(t, f, root2, proof4, "acdcda", "1", true)
}

// =================
// helper funcs for working with keys represented by hex value strings
// used when the general relations between keys are length wise
// =================
func hexStringToBytes(s string) []byte {
	if (len(s) % 2) != 0 {
		panic("key value needs to be a hex representation of a byte array")
	}
	bytesKey := make([]byte, len(s)/2)
	hex.Decode(bytesKey, []byte(s))
	return bytesKey
}

func generalKeyUpdateEntries(f *Forest, baseHash primitives.Sha256, keyValues ...string) primitives.Sha256 {
	if len(keyValues)%2 != 0 {
		panic("expected key value pairs")
	}
	diffs := make(TrieDiffs, len(keyValues)/2)
	for i := 0; i < len(keyValues); i = i + 2 {
		diffs[i/2] = &TrieDiff{Key: hexStringToBytes(keyValues[i]), Value: hash.CalcSha256([]byte(keyValues[i+1]))}
	}

	currentRoot, _ := f.Update(baseHash, diffs)

	return currentRoot
}

func generalKeyVerifyProof(t *testing.T, f *Forest, root primitives.Sha256, proof TrieProof, path string, value string, exists bool) {
	verifyProof(t, f, root, proof, hexStringToBytes(path), value, exists)
}

func generalKeyGetProofRequireHeight(t *testing.T, f *Forest, root primitives.Sha256, path string, expectedHeight int) TrieProof {
	return getProofRequireHeight(t, f, root, hexStringToBytes(path), expectedHeight)
}

func generalKeyGetProof(t *testing.T, f *Forest, root primitives.Sha256, path string) TrieProof {
	proof, err := f.GetProof(root, hexStringToBytes(path))
	require.NoError(t, err, "failed with error: %s", err)
	return proof
}


// =================
// helper funcs for working with keys represented by a string with 0s or 1s
// used when the specific node relations are required in tests
// =================
func isChar0(ch uint8) byte {
	if ch == uint8('0') {
		return 0
	}
	return 1
}

func bitStringToBytes(s string) []byte {
	if (len(s) % 8) != 0 {
		panic("key value needs to be a bit representation of a byte array")
	}
	bytelen := len(s) / 8
	bytesKey := make([]byte, bytelen)
	for i := 0; i < bytelen; i++ {
		bytesKey[i] = isChar0(s[i*8])<<7 |
			isChar0(s[i*8+1])<<6 |
			isChar0(s[i*8+2])<<5 |
			isChar0(s[i*8+3])<<4 |
			isChar0(s[i*8+4])<<3 |
			isChar0(s[i*8+5])<<2 |
			isChar0(s[i*8+6])<<1 |
			isChar0(s[i*8+7])
	}
	return bytesKey
}

func binaryKeyUpdateEntries(f *Forest, baseHash primitives.Sha256, keyValues ...string) primitives.Sha256 {
	if len(keyValues)%2 != 0 {
		panic("expected key value pairs")
	}
	diffs := make(TrieDiffs, len(keyValues)/2)
	for i := 0; i < len(keyValues); i = i + 2 {
		diffs[i/2] = &TrieDiff{Key: bitStringToBytes(keyValues[i]), Value: hash.CalcSha256([]byte(keyValues[i+1]))}
	}

	currentRoot, _ := f.Update(baseHash, diffs)

	return currentRoot
}

func binaryKeyVerifyProof(t *testing.T, f *Forest, root primitives.Sha256, proof TrieProof, path string, value string, exists bool) {
	verifyProof(t, f, root, proof, bitStringToBytes(path), value, exists)
}

func binaryKeyGetProofRequireHeight(t *testing.T, f *Forest, root primitives.Sha256, path string, expectedHeight int) TrieProof {
	return getProofRequireHeight(t, f, root, bitStringToBytes(path), expectedHeight)
}

// =================
// verify and getproof short hand funcs
// =================
func verifyProof(t *testing.T, f *Forest, root primitives.Sha256, proof TrieProof, path []byte, value string, exists bool) {
	verified, err := f.Verify(root, proof, path, hash.CalcSha256([]byte(value)))
	require.NoError(t, err, "proof verification failed")
	require.Equal(t, exists, verified, "proof verification returned unexpected result")
}

func getProofRequireHeight(t *testing.T, f *Forest, root primitives.Sha256, path []byte, expectedHeight int) TrieProof {
	proof, err := f.GetProof(root, path)
	require.NoError(t, err, "failed with error: %s", err)
	require.Equal(t, expectedHeight, len(proof), "unexpected proof length")
	return proof
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
	if n.left != nil {
		n.left.printNode("0", depth+len(pathString)-1, trie, t)
	}
	if n.right != nil {
		n.right.printNode("1", depth+len(pathString)-1, trie, t)
	}
}
