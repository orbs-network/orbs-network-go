// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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

/***
NOTE : merkle trie proofs assume a fixed size key for all entries. So make sure for each test all entries have same size key.
***/

func TestTrieNodeHashFunc_Hash(t *testing.T) {
	res := make([][]byte, 3)
	res[0] = make([]byte, hash.SHA256_HASH_SIZE_BYTES)
	res[0][10] = 16
	res[1] = make([]byte, hash.SHA256_HASH_SIZE_BYTES)
	res[1][20] = 16
	hash1 := hash.CalcSha256(res...)
	res[2] = make([]byte, hash.SHA256_HASH_SIZE_BYTES)
	hash2 := hash.CalcSha256(res...)
	require.NotEqual(t, hash1, hash2, "should not be equal")
	hash3 := hash.CalcSha256(res[0], res[1])
	require.Equal(t, hash1, hash3, "should be equal")
}

func TestTrieNodeHashFunc_Table(t *testing.T) {
	leftPrefix := []byte{1, 0, 0, 0}
	rightPrefix := []byte{0, 0, 0, 1}
	leftValue := hash.CalcSha256([]byte("left"))
	rightValue := hash.CalcSha256([]byte("right"))
	leftHash := hash.CalcSha256(leftValue, leftPrefix)
	rightHash := hash.CalcSha256(rightValue, rightPrefix)
	leftLeaf := &node{path: leftPrefix, value: leftValue, hash: leftHash}
	rightLeaf := &node{path: rightPrefix, value: rightValue, hash: rightHash}
	tests := []struct {
		name string
		n    *node
	}{
		{"empty leaf node", &node{[]byte{}, primitives.Sha256{}, primitives.Sha256{}, nil, nil}},
		{"leaf node", &node{path: leftPrefix, value: leftValue}},
		{"node with left", &node{path: []byte{1, 1, 1, 1}, left: leftLeaf, right: nil}},
		{"node with left no prefix", &node{path: []byte{}, left: leftLeaf, right: nil}},
		{"node with right", &node{path: []byte{1, 1, 1, 1}, left: nil, right: rightLeaf}},
		{"node with both", &node{path: []byte{1, 1, 1, 1}, left: leftLeaf, right: rightLeaf}},
	}
	for i := range tests {
		cTest := tests[i] // this is so that we can run tests in parallel, see https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721
		t.Run(cTest.name, func(t *testing.T) {
			t.Parallel()
			treeHash := hashTrieNode(cTest.n)
			testHash := recHashCalc(cTest.n)
			require.Equal(t, testHash, treeHash, "%s proof node size mismatch", cTest.name)
		})
	}
}

func recHashCalc(n *node) primitives.Sha256 {
	if n == nil {
		return make([]byte, hash.SHA256_HASH_SIZE_BYTES)
	} else if n.isLeaf() {
		return hash.CalcSha256(n.value, n.path)
	} else {
		return hash.CalcSha256(recHashCalc(n.left), recHashCalc(n.right), n.path)
	}
}

func TestRoot_Management(t *testing.T) {
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

func TestRoot_ForgetWhenMultipleSameRootsAndKeepOrder(t *testing.T) {
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

func TestRoot_UpdateTrieFailsForMissingRoot(t *testing.T) {
	f, _ := NewForest()
	badroot := hash.CalcSha256([]byte("deaddead"))

	root := updateEntries(f, badroot, "abcdef", "val")

	require.Nil(t, root, "did not receive an empty response when using a corrupt merkle root")
}

func TestProof_ValidationForNonCompatibleKey(t *testing.T) {
	f, root := NewForest()
	key := "deaddead"
	proof := getProofRequireHeight(t, f, root, key, 0)
	verifyProof(t, f, root, proof, key, "", true)
	verifyProof(t, f, root, proof, key, "non-zero", false)

	root1 := updateEntries(f, root, "abcdef01", "val")
	proof1 := getProofRequireHeight(t, f, root1, key, 1)
	verifyProof(t, f, root1, proof1, key, "", true)
	verifyProof(t, f, root1, proof1, key, "non-zero", false)
}

func TestProof_ValidationForTwoRevisionsOfSameKey(t *testing.T) {
	f, root := NewForest()
	root1 := updateEntries(f, root, "abc1", "baz1")
	root2 := updateEntries(f, root1, "abc1", "baz2")

	proof1 := getProofRequireHeight(t, f, root1, "abc1", 1)
	verifyProof(t, f, root1, proof1, "abc1", "baz1", true)

	proof2 := getProofRequireHeight(t, f, root2, "abc1", 1)
	verifyProof(t, f, root2, proof2, "abc1", "baz2", true)

	require.NotEqual(t, proof1, proof2, "proofs are different")
	require.Equal(t, len(proof1.nodes), len(proof2.nodes), "proofs are equal length")
	require.Equal(t, 16, proof2.nodes[0].prefixSize, "proofs are equal length")
}

func TestProof_ValidationForSimpleBranchingTrie(t *testing.T) {
	f, root := NewForest()
	root1 := updateEntries(f, root, "abc1", "baz1", "abd1", "baz2")

	proof := getProofRequireHeight(t, f, root1, "abc1", 2)
	require.EqualValues(t, 11, proof.nodes[0].prefixSize, "proof node 0, wring prefix size")
	require.EqualValues(t, 4, proof.nodes[1].prefixSize, "proof node 1, wring prefix size")
	// verify with correct key
	verifyProof(t, f, root1, proof, "abc1", "baz1", true)
	verifyProof(t, f, root1, proof, "abc1", "baz2", false)
	verifyProof(t, f, root1, proof, "abc1", "", false) // since it actually exists then its NOT excluded
	// verify with wrong key that is diff only in leaf - exclusion possible
	verifyProof(t, f, root1, proof, "abc6", "baz1", false)
	verifyProof(t, f, root1, proof, "abc6", "baz2", false)
	verifyProof(t, f, root1, proof, "abc6", "", true)
	// verify with wrong key that is diff above leaf - proof inconsistent
	verifInconsistentProof(t, f, root1, proof, "abd1", "baz1")
	verifInconsistentProof(t, f, root1, proof, "abd1", "baz4")
	verifInconsistentProof(t, f, root1, proof, "abd1", "")

	proof2 := getProofRequireHeight(t, f, root1, "abc2", 2)
	require.EqualValues(t, 11, proof2.nodes[0].prefixSize, "proof2 node 0, wring prefix size")
	require.EqualValues(t, 4, proof2.nodes[1].prefixSize, "proof2 node 1, wring prefix size")
	verifyProof(t, f, root1, proof2, "abc2", "", true)
	verifyProof(t, f, root1, proof2, "abc2", "baz1", false)
	verifyProof(t, f, root1, proof2, "abc1", "baz1", true) // unlikely to happen in real life to guess correct key and value
	// verify with wrong key that is diff only in leaf - exclusion possible
	verifyProof(t, f, root1, proof2, "abc6", "baz1", false)
	verifyProof(t, f, root1, proof2, "abc6", "baz2", false)
	verifyProof(t, f, root1, proof2, "abc6", "", true)
	// verify with wrong key that is diff above leaf - proof inconsistent
	verifInconsistentProof(t, f, root1, proof, "acd1", "baz1")
	verifInconsistentProof(t, f, root1, proof, "acd1", "")
}

func TestProof_ValidationAfterUpdateWithBothReplaceAndNewValue(t *testing.T) {
	f, root := NewForest()

	root1 := updateEntries(f, root, "abc1", "baz", "1234", "quux1")

	proof := getProofRequireHeight(t, f, root1, "abc1", 2)
	verifyProof(t, f, root1, proof, "abc1", "baz", true)
	proof = getProofRequireHeight(t, f, root1, "abc2", 2)
	verifyProof(t, f, root1, proof, "abc2", "qux", false)

	root2 := updateEntries(f, root1, "abc2", "qux", "1234", "quux2")
	require.NotEqual(t, root2, root1, "roots should be different")

	// retest that first insert proof still valid
	proof = getProofRequireHeight(t, f, root1, "abc1", 2)
	verifyProof(t, f, root1, proof, "abc1", "baz", true)
	proof = getProofRequireHeight(t, f, root1, "abc2", 2)
	verifyProof(t, f, root1, proof, "abc2", "qux", false)
	proof = getProofRequireHeight(t, f, root1, "1234", 2)
	verifyProof(t, f, root1, proof, "1234", "quux1", true)
	verifyProof(t, f, root1, proof, "1234", "quux2", false)

	// after second insert proofs
	proof = getProofRequireHeight(t, f, root2, "abc2", 3)
	verifyProof(t, f, root2, proof, "abc2", "qux", true)
	proof = getProofRequireHeight(t, f, root2, "abc1", 3)
	verifyProof(t, f, root2, proof, "abc1", "baz", true)
	proof = getProofRequireHeight(t, f, root2, "1234", 2)
	verifyProof(t, f, root2, proof, "1234", "quux2", true)
	verifyProof(t, f, root2, proof, "1234", "quux1", false)
}

func TestProof_ReplaceExistingValueBelowDivergingPaths(t *testing.T) {
	f, root := NewForest()
	root1 := updateEntries(f, root, "0001", "baz", "0000", "qux", "00a0", "bar", "00e0", "quux")
	root2 := updateEntries(f, root1, "0000", "zoo")

	proof := getProofRequireHeight(t, f, root2, "0000", 3)
	verifyProof(t, f, root2, proof, "0000", "zoo", true)
	verifyProof(t, f, root2, proof, "0000", "qux", false)
}

func TestProof_ValidationForTwoLevelsBranchingTrie(t *testing.T) {
	f, root := NewForest()
	root1 := updateEntries(f, root, "0000", "baz1", "0100", "baz2", "0110", "baz3")

	// on left (short proof)
	proof1 := getProofRequireHeight(t, f, root1, "0000", 2)
	require.EqualValues(t, 7, proof1.nodes[0].prefixSize, "proof1 node 0, wring prefix size")
	require.EqualValues(t, 8, proof1.nodes[1].prefixSize, "proof1 node 1, wring prefix size")
	verifyProof(t, f, root1, proof1, "0000", "baz1", true)
	verifyProof(t, f, root1, proof1, "0000", "", false) // since it actually exists then its NOT excluded
	// verify with wrong key that is diff in core node - proof is inconsistent
	verifInconsistentProof(t, f, root1, proof1, "0100", "baz1")
	verifInconsistentProof(t, f, root1, proof1, "0100", "")
	verifInconsistentProof(t, f, root1, proof1, "1000", "baz1")
	// verify with wrong key that is diff only in leaf - exclusion possible
	verifyProof(t, f, root1, proof1, "0001", "baz3", false)
	verifyProof(t, f, root1, proof1, "0001", "", true)

	// on right (long proof)
	proof2 := getProofRequireHeight(t, f, root1, "0110", 3)
	require.EqualValues(t, 7, proof2.nodes[0].prefixSize, "proof2 node 0, wring prefix size")
	require.EqualValues(t, 3, proof2.nodes[1].prefixSize, "proof2 node 1, wring prefix size")
	require.EqualValues(t, 4, proof2.nodes[2].prefixSize, "proof2 node 2, wring prefix size")
	verifyProof(t, f, root1, proof2, "0110", "baz3", true)
	verifyProof(t, f, root1, proof2, "0110", "", false) // since it actually exists then its NOT excluded
	// verify with wrong key that is diff in core node - proof is inconsistent
	verifInconsistentProof(t, f, root1, proof2, "0100", "baz3") // wrong key above split
	verifInconsistentProof(t, f, root1, proof2, "0120", "baz3") // wrong key above split
	// verify with wrong key that is diff only in leaf - exclusion possible
	verifyProof(t, f, root1, proof2, "0111", "baz3", false)
	verifyProof(t, f, root1, proof2, "0111", "", true)
}

func TestProof_ValidationForKeyNotInTreeTwoLevelsBranchingTrie(t *testing.T) {
	f, root := NewForest()
	root1 := updateEntries(f, root, "0000", "baz1", "0100", "baz2", "0110", "baz3")

	// on left (short proof)
	proof1 := getProofRequireHeight(t, f, root1, "0001", 2)
	require.EqualValues(t, 7, proof1.nodes[0].prefixSize, "proof1 node 0, wring prefix size")
	require.EqualValues(t, 8, proof1.nodes[1].prefixSize, "proof1 node 1, wring prefix size")
	verifyProof(t, f, root1, proof1, "0001", "", true)
	verifyProof(t, f, root1, proof1, "0001", "baz1", false)
	// verify with wrong key that is diff in core node - proof is inconsistent
	verifInconsistentProof(t, f, root1, proof1, "0100", "baz1")
	// verify with wrong key that is diff in leaf node - exclusion possible
	verifyProof(t, f, root1, proof1, "0002", "baz1", false)
	verifyProof(t, f, root1, proof1, "0002", "", true)

	// on right (long proof)
	proof2 := getProofRequireHeight(t, f, root1, "0111", 3)
	require.EqualValues(t, 7, proof2.nodes[0].prefixSize, "proof2 node 0, wring prefix size")
	require.EqualValues(t, 3, proof2.nodes[1].prefixSize, "proof2 node 1, wring prefix size")
	require.EqualValues(t, 4, proof2.nodes[2].prefixSize, "proof2 node 2, wring prefix size")
	verifyProof(t, f, root1, proof2, "0111", "baz3", false)
	verifyProof(t, f, root1, proof2, "0111", "", true)
	// verify with wrong key that is diff in core node - proof is inconsistent
	verifInconsistentProof(t, f, root1, proof2, "0200", "")
	// verify with wrong key that is diff in leaf node - exclusion possible
	verifyProof(t, f, root1, proof2, "0112", "baz3", false)
	verifyProof(t, f, root1, proof2, "0112", "", true)
}

func TestProof_ValidationForMissingKeyTwoLevelsBranchingTrie(t *testing.T) {
	f, root := NewForest()
	root1 := updateEntries(f, root, "0000", "baz1", "0100", "baz2", "0110", "baz3")

	key := "0111" // under the second branch
	proof := getProofRequireHeight(t, f, root1, key, 3)
	verifyProof(t, f, root1, proof, key, "", true)
	verifyProof(t, f, root1, proof, key, "non-zero", false)

	key2 := "0011" // under the first branch
	proof2 := getProofRequireHeight(t, f, root1, key2, 2)
	verifyProof(t, f, root1, proof2, key2, "", true)
	verifyProof(t, f, root1, proof2, key2, "non-zero", false)
}

func TestProof_ValidationForMissingKeyDivergentInMiddle(t *testing.T) {
	f, root := NewForest()
	root1 := updateEntries(f, root, "00000000", "baz1", "00100000", "baz2", "00000111", "baz3")

	key := "00001111" // mismatch in the second branch == middle
	proof := getProofRequireHeight(t, f, root1, key, 2)
	verifyProof(t, f, root1, proof, key, "", true)
	verifyProof(t, f, root1, proof, key, "non-zero", false)
	// verify with wrong key that is diff in core node - proof is inconsistent
	verifInconsistentProof(t, f, root1, proof, "02000000", "")
	// verify with wrong key that is diff in leaf node - exclusion possible
	verifyProof(t, f, root1, proof, "00000011", "baz3", false)
	verifyProof(t, f, root1, proof, "00000011", "", true)
}

func TestProof_ValidationKeyWithLeavesWithNoPrefix(t *testing.T) {
	f, root := NewForest()
	root1 := updateEntries(f, root, "0000", "baz1", "0001", "baz2", "0100", "baz3", "0101", "baz4")

	key := "0001"
	proof := getProofRequireHeight(t, f, root1, key, 3)
	verifyProof(t, f, root1, proof, key, "baz2", true)
	verifyProof(t, f, root1, proof, key, "baz3", false)
	verifyProof(t, f, root1, proof, key, "", false)
	// verify with wrong key that is diff in core node - proof is inconsistent
	verifInconsistentProof(t, f, root1, proof, "1000", "")
}

func TestProof_OrderOfAdditionsDoesNotMatter(t *testing.T) {
	keyValue := []string{"abcd1234", "baz", "abc12300", "qux", "abc12345", "quux1234", "aadd1234", "foo", "12345678", "hello"}
	var1 := []int{2, 6, 0, 8, 4}
	var2 := []int{8, 4, 0, 2, 6}
	var3 := []int{8, 6, 4, 2, 0}

	f1, initRoot1 := NewForest()
	root1 := updateEntries(f1, initRoot1, keyValue[var1[0]], keyValue[var1[0]+1], keyValue[var1[1]], keyValue[var1[1]+1],
		keyValue[var1[2]], keyValue[var1[2]+1], keyValue[var1[3]], keyValue[var1[3]+1], keyValue[var1[4]], keyValue[var1[4]+1])
	proof1 := getProof(t, f1, root1, "abc12345")

	f2, initRoot2 := NewForest()
	root2 := updateEntries(f2, initRoot2, keyValue[var2[0]], keyValue[var2[0]+1], keyValue[var2[1]], keyValue[var2[1]+1],
		keyValue[var2[2]], keyValue[var2[2]+1], keyValue[var2[3]], keyValue[var2[3]+1], keyValue[var2[4]], keyValue[var2[4]+1])
	proof2 := getProof(t, f2, root2, "abc12345")

	require.Equal(t, root1, root2, "unexpected different root hash")
	require.Equal(t, len(proof1.nodes), len(proof2.nodes), "unexpected different tree depth / proof lengths")
	require.Equal(t, proof1.nodes[3].otherChildHash, proof2.nodes[3].otherChildHash, "unexpected different leaf node hash")

	f3, initRoot3 := NewForest()
	root3 := updateEntries(f3, initRoot3, keyValue[var3[0]], keyValue[var3[0]+1], keyValue[var3[1]], keyValue[var3[1]+1],
		keyValue[var3[2]], keyValue[var3[2]+1], keyValue[var3[3]], keyValue[var3[3]+1], keyValue[var3[4]], keyValue[var3[4]+1])
	proof3 := getProof(t, f3, root3, "abc12345")

	require.Equal(t, len(proof2.nodes), len(proof3.nodes), "unexpected different tree depth / proof lengths")
	require.Equal(t, proof2.nodes[3].otherChildHash, proof3.nodes[3].otherChildHash, "unexpected different leaf node hash")
}

func TestProof_AddConvegingPathsWithExactValues(t *testing.T) {
	f, root := NewForest()
	root1 := updateEntries(f, root, "abdbda", "1", "abdcda", "1", "acdbda", "1", "acdcda", "1")
	root2 := updateEntries(f, root1, "abdcda", "2")

	proof1 := getProof(t, f, root2, "abdbda")
	proof2 := getProof(t, f, root2, "abdcda")
	proof3 := getProof(t, f, root2, "acdbda")
	proof4 := getProof(t, f, root2, "acdcda")

	verifyProof(t, f, root2, proof1, "abdbda", "1", true)
	verifyProof(t, f, root2, proof2, "abdcda", "2", true)
	verifyProof(t, f, root2, proof3, "acdbda", "1", true)
	verifyProof(t, f, root2, proof4, "acdcda", "1", true)
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

func updateEntries(f *Forest, baseHash primitives.Sha256, keyValues ...string) primitives.Sha256 {
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

func verifyProof(t *testing.T, f *Forest, root primitives.Sha256, proof *TrieProof, path string, value string, exists bool) {
	verified, err := f.Verify(root, proof, hexStringToBytes(path), hash.CalcSha256([]byte(value)))
	require.NoError(t, err, "proof verification failed")
	require.Equal(t, exists, verified, "proof verification returned unexpected result")
}

func verifInconsistentProof(t *testing.T, f *Forest, root primitives.Sha256, proof *TrieProof, path string, value string) {
	_, err := f.Verify(root, proof, hexStringToBytes(path), hash.CalcSha256([]byte(value)))
	require.Error(t, err, "proof should have failed")
}

func getProofRequireHeight(t *testing.T, f *Forest, root primitives.Sha256, path string, expectedHeight int) *TrieProof {
	proof, err := f.GetProof(root, hexStringToBytes(path))
	require.NoError(t, err, "failed with error: %s", err)
	require.Equal(t, expectedHeight, len(proof.nodes), "unexpected proof length")
	return proof
}

func getProof(t *testing.T, f *Forest, root primitives.Sha256, path string) *TrieProof {
	proof, err := f.GetProof(root, hexStringToBytes(path))
	require.NoError(t, err, "failed with error: %s", err)
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
