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

// TODO test the hashes?
//func TestTrieNodeProof(t *testing.T) {
//	leftleft := &node{hash: hash.CalcSha256([]byte("left"))}
//	rightleaf := &node{hash: hash.CalcSha256([]byte("right"))}
//	tests := []struct {
//		name         string
//		n            *node
//		expectedSize int
//	}{
//		{"empty leaf node", &node{[]byte{}, primitives.Sha256{}, primitives.Sha256{}, nil, nil}, 2},
//		{"leaf node", &node{value: hash.CalcSha256([]byte("value"))}, 2},
//		{"node with left", &node{[]byte{}, primitives.Sha256{}, primitives.Sha256{}, leftleft, nil}, 3},
//		{"node with right", &node{[]byte{}, primitives.Sha256{}, primitives.Sha256{}, nil, rightleaf}, 3},
//		{"node with both", &node{[]byte{}, primitives.Sha256{}, primitives.Sha256{}, leftleft, rightleaf}, 3},
//	}
//	for i := range tests {
//		cTest := tests[i] // this is so that we can run tests in parallel, see https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721
//		t.Run(cTest.name, func(t *testing.T) {
//			t.Parallel()
//			trieProof := generateNodeOrLeafProof(cTest.n)
//			require.Equal(t, cTest.expectedSize, len(trieProof), "%s proof node size mismatch", cTest.name)
//			//			require.Equal(t, cTest.keysize, tree.keySize, "tree max depth size error", cTest.name)
//		})
//	}
//}

func TestFromBin(t *testing.T) {
	tests := []struct {
		name  string
		bin   []byte
		bytes [32]byte
	}{
		{"empty", []byte{}, [32]byte{}},
		{"small non empty zero", []byte{0, 0, 0, 0, 0}, [32]byte{}},
		{"large non empty zero", []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, [32]byte{}},
		{"small 1", []byte{0, 0, 0, 0, 0, 0, 0, 1}, [32]byte{1}},
		{"large 1", []byte{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, [32]byte{1}},
		{"small 16", []byte{0, 0, 0, 1}, [32]byte{16}},
		{"large 16", []byte{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, [32]byte{16}},
		{"small2 2", []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, [32]byte{0, 2}},
		{"large2 2", []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, [32]byte{0, 2}},
		{"large3", []byte{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0}, [32]byte{1, 2, 5, 9}},
	}
	for i := range tests {
		cTest := tests[i] // this is so that we can run tests in parallel, see https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721
		t.Run(cTest.name, func(t *testing.T) {
			t.Parallel()
			b := fromBin(cTest.bin)
			require.Len(t, b, 32, "%s failed size", cTest.name)
			for i := 0; i < 32; i++ {
				require.Equal(t, cTest.bytes[i], b[i], "%s failed at byte %d", cTest.name, i)
			}
		})
	}
}

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

func TestValidProofForMissingKey(t *testing.T) {
	f, root := NewForest()
	key := "deaddead"
	proof := generalKeyGetProofRequireHeight(t, f, root, key, 1)
	generalKeyVerifyProof(t, f, root, proof, key, "", true)
	generalKeyVerifyProof(t, f, root, proof, key, "non-zero", false)

	root1 := generalKeyUpdateEntries(f, root, "abcdef", "val")
	proof = generalKeyGetProofRequireHeight(t, f, root1, key, 1)
	generalKeyVerifyProof(t, f, root1, proof, key, "", true)
	generalKeyVerifyProof(t, f, root1, proof, key, "non-zero", false)
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

func TestProofReplaceExistingValueBelowDivergingPaths(t *testing.T) {
	f, root := NewForest()
	root1 := binaryKeyUpdateEntries(f, root, "00000000", "baz", "0000000000000000", "qux", "0000000010000000", "bar", "0000000011000000", "quux")
	root2 := binaryKeyUpdateEntries(f, root1, "0000000000000000", "zoo")

	proof := binaryKeyGetProofRequireHeight(t, f, root2, "0000000000000000", 2)
	binaryKeyVerifyProof(t, f, root2, proof, "0000000000000000", "zoo", true)
	binaryKeyVerifyProof(t, f, root2, proof, "0000000000000000", "qux", false)
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

// TODO make sure that prefixSize tested for stuff like the next two
//func TestRemoveValue_ParentWithSingleChild(t *testing.T) {
//	f, root := NewForest()
//
//	root1 := generalKeyUpdateEntries(f, root, "ab", "1", "abcd", "1")
//	root2 := generalKeyUpdateEntries(f, root1, "ab", "")
//
//	require.EqualValues(t, "1010101111001101" /*"abcd"*/, bytesToBinaryString(f.findRoot(root2).path), "expected root path doesn't fit actual tree")
//
//	p := generalKeyGetProofRequireHeight(t, f, root2, "abcd", 1)
//	require.Len(t, "1010101111001101" /*"abcd"*/, p[0].prefixSize, "full tree proof for and does not end with expected node path")
//}
//func TestRemoveValue_NonBranchingNonLeaf1(t *testing.T) {
//	f, root := NewForest()
//
//	fullTree := generalKeyUpdateEntries(f, root, "ab", "1", "abcd", "2", "abcdef", "3")
//	require.EqualValues(t, "1001101" /* "cd" without first bit*/, bytesToBinaryString(f.findRoot(fullTree).right.path), "before remove right child path is not correct")
//	afterRemove := generalKeyUpdateEntries(f, fullTree, "abcd", "")
//	require.EqualValues(t, "100110111101111" /*"cdef" without first bit*/, bytesToBinaryString(f.findRoot(afterRemove).right.path), "after remove right child path is not correct")
//
//	p1 := generalKeyGetProofRequireHeight(t, f, fullTree, "abcd", 2)
//	p2 := generalKeyGetProofRequireHeight(t, f, afterRemove, "abcd", 2)
//
//	generalKeyGetProofRequireHeight(t, f, fullTree, "abcdef", 3)
//	generalKeyGetProofRequireHeight(t, f, afterRemove, "abcdef", 2)
//
//	require.Len(t, "1001101" /* "cd" without first bit*/, p1[1].prefixSize, "full tree proof for and does not end with expected node path")
//	require.Len(t, "100110111101111" /*"cdef" without first bit*/, p2[1].prefixSize, "full tree proof for and does not end with expected node path")
//}

func TestProofOrderOfAdditionsDoesNotMatter(t *testing.T) {
	keyValue := []string{"abcd1234", "baz", "abc12300", "qux", "abc12345", "quux1234", "aadd1234", "foo", "12345678", "hello"}
	var1 := []int{2, 6, 0, 8, 4}
	var2 := []int{8, 4, 0, 2, 6}
	var3 := []int{8, 6, 4, 2, 0}

	f1, initRoot1 := NewForest()
	root1 := generalKeyUpdateEntries(f1, initRoot1, keyValue[var1[0]], keyValue[var1[0]+1], keyValue[var1[1]], keyValue[var1[1]+1],
		keyValue[var1[2]], keyValue[var1[2]+1], keyValue[var1[3]], keyValue[var1[3]+1], keyValue[var1[4]], keyValue[var1[4]+1])
	proof1 := generalKeyGetProof(t, f1, root1, "abc12345")

	f2, initRoot2 := NewForest()
	root2 := generalKeyUpdateEntries(f2, initRoot2, keyValue[var2[0]], keyValue[var2[0]+1], keyValue[var2[1]], keyValue[var2[1]+1],
		keyValue[var2[2]], keyValue[var2[2]+1], keyValue[var2[3]], keyValue[var2[3]+1], keyValue[var2[4]], keyValue[var2[4]+1])
	proof2 := generalKeyGetProof(t, f2, root2, "abc12345")

	require.Equal(t, root1, root2, "unexpected different root hash")
	require.Equal(t, len(proof1), len(proof2), "unexpected different tree depth / proof lengths")
	require.Equal(t, proof1[3].hashValue, proof2[3].hashValue, "unexpected different leaf node hash")

	f3, initRoot3 := NewForest()
	root3 := generalKeyUpdateEntries(f3, initRoot3, keyValue[var3[0]], keyValue[var3[0]+1], keyValue[var3[1]], keyValue[var3[1]+1],
		keyValue[var3[2]], keyValue[var3[2]+1], keyValue[var3[3]], keyValue[var3[3]+1], keyValue[var3[4]], keyValue[var3[4]+1])
	proof3 := generalKeyGetProof(t, f3, root3, "abc12345")

	require.Equal(t, len(proof2), len(proof3), "unexpected different tree depth / proof lengths")
	require.Equal(t, proof2[3].hashValue, proof3[3].hashValue, "unexpected different leaf node hash")
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
