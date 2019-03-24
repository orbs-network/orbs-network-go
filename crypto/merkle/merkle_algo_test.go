// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package merkle

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAddSingleEntryToEmptyTree(t *testing.T) {
	root := genOrUpdateTree(nil, "0101", "baz")

	requireIsNode(t, root, "0101", "baz", false, false)
}

func TestRootChangeAfterStateChange(t *testing.T) {
	root1 := genOrUpdateTree(nil, "010101", "baz")
	root2 := genOrUpdateTree(root1, "010101", "baz2")

	requireIsNode(t, root2, "010101", "baz2", false, false)
	require.NotEqual(t, root1.hash, root2.hash, "root hash did not change after state change")
}

func TestRevertingStateChangeRevertsMerkleRoot(t *testing.T) {
	root1 := genOrUpdateTree(nil, "010101", "baz")
	root2 := genOrUpdateTree(root1, "010101", "baz2")
	root3 := genOrUpdateTree(root2, "010101", "baz")

	require.Equal(t, root1.hash, root3.hash, "root hash did not revert back after resetting state")
}

func TestExtendingLeafNode(t *testing.T) {
	// one at a time
	root11 := genOrUpdateTree(nil, "01", "zoo")
	root12 := genOrUpdateTree(root11, "0101", "baz")
	requireIsNode(t, root12, "01", "zoo", true, false)
	requireIsNode(t, root12.left, "1", "baz", false, false)
	root13 := genOrUpdateTree(root12, "010101", "Hello")
	requireIsNode(t, root13, "01", "zoo", true, false)
	requireIsNode(t, root13.left, "1", "baz", true, false)
	requireIsNode(t, root13.left.left, "1", "Hello", false, false)

	// all together
	root23 := genOrUpdateTree(nil, "01", "zoo", "0101", "baz", "010101", "Hello")

	// all together diff order
	root33 := genOrUpdateTree(nil, "010101", "Hello", "0101", "baz", "01", "zoo")

	require.Equal(t, root13.hash, root23.hash, "should be same root")
	require.Equal(t, root13.hash, root33.hash, "should be same root")
}

func TestExtendingKeyPathBySeveralChars(t *testing.T) {
	root1 := genOrUpdateTree(nil, "01", "baz", "0111", "qux", "011111111111", "quux")
	requireIsNode(t, root1.right, "1", "qux", false, true)
	requireIsNode(t, root1.right.right, "1111111", "quux", false, false)
}

func TestUpdateWithBothReplaceAndNewValue(t *testing.T) {
	root1 := genOrUpdateTree(nil, "0000", "baz", "1111", "quux1")
	requireIsNode(t, root1, "", "", true, true)

	root2 := genOrUpdateTree(root1, "1111", "quux2", "0100", "qux")
	requireIsNode(t, root2.right, "111", "quux2", false, false)
	requireIsNode(t, root2.left, "", "", true, true)
	requireIsNode(t, root2.left.left, "00", "baz", false, false)
	requireIsNode(t, root2.left.right, "00", "qux", false, false)
}

func TestAddSiblingNode(t *testing.T) {
	root1 := genOrUpdateTree(nil, "00000000", "baz")
	root2 := genOrUpdateTree(root1, "0000000000000000", "qux", "0000000010000000", "quux")
	requireIsNode(t, root2, "00000000", "baz", true, true)
	requireIsNode(t, root2.left, "0000000", "qux", false, false)
	requireIsNode(t, root2.right, "0000000", "quux", false, false)
}

func TestAddPathToCauseBranchingAlongExistingPath(t *testing.T) {
	root1 := genOrUpdateTree(nil, "00000000", "baz", "0000000000000000", "qux")
	root2 := genOrUpdateTree(root1, "0000000010000000", "quux")
	requireIsNode(t, root2, "00000000", "baz", true, true)
	requireIsNode(t, root2.left, "0000000", "qux", false, false)
	requireIsNode(t, root2.right, "0000000", "quux", false, false)
}

func TestReplaceExistingValueBelowDivergingPaths(t *testing.T) {
	root1 := genOrUpdateTree(nil, "00000000", "baz", "0000000000000000", "qux", "0000000010000000", "bar", "0000000011000000", "quux")
	root2 := genOrUpdateTree(root1, "0000000000000000", "zoo")
	requireIsNode(t, root2.left, "0000000", "zoo", false, false)
}

func TestAddPathToCauseNewParent(t *testing.T) {
	root1 := genOrUpdateTree(nil, "001100", "Hirsch", "0011", "Hello")

	requireIsNode(t, root1, "0011", "Hello", true, false)
	requireIsNode(t, root1.left, "0", "Hirsch", false, false)
}

func TestRemoveValue_SingleExistingNode(t *testing.T) {
	root1 := genOrUpdateTree(nil, "01", "aValue")
	root2 := genOrUpdateTree(root1, "01", "")

	requireIsNode(t, root1, "01", "aValue", false, false)
	requireIsNode(t, root2, "", "", false, false)

	require.EqualValues(t, createEmptyNode().hash, root2.hash, "for identical states hash must be identical")
	require.NotEqual(t, root1.hash, root2.hash, "for different states hash must be different")
}

func TestRemoveValue_RemoveSingleChildLeaf(t *testing.T) {
	root1 := genOrUpdateTree(nil, "0000", "1")
	root2 := genOrUpdateTree(root1, "000011", "2")
	root3 := genOrUpdateTree(root2, "000011", "")

	requireIsNode(t, root3, "0000", "1", false, false)
	require.EqualValues(t, root1.hash, root3.hash, "root hash should be identical")
}

func TestRemoveValue_ParentWithSingleChild(t *testing.T) {
	root1 := genOrUpdateTree(nil, "00", "1", "0011", "2")
	root2 := genOrUpdateTree(root1, "00", "")

	requireIsNode(t, root2, "0011", "2", false, false)
}

func TestRemoveValue_NonBranchingNonLeaf1(t *testing.T) {
	fullTree := genOrUpdateTree(nil, "00", "1", "0000", "2", "000000", "3")
	afterRemove := genOrUpdateTree(fullTree, "0000", "")

	requireIsNode(t, afterRemove.left, "000", "3", false, false)
}

func TestRemoveValue_BranchingNonLeaf_NodeStructureUnchanged(t *testing.T) {
	fullTree := genOrUpdateTree(nil, "00", "1", "0000", "2", "0011", "3")
	afterRemove := genOrUpdateTree(fullTree, "00", "")
	requireIsNode(t, afterRemove, "00", "", true, true)
	requireIsNode(t, afterRemove.left, "0", "2", false, false)
	requireIsNode(t, afterRemove.right, "1", "3", false, false)
}

func TestRemoveValue_BranchingNonLeaf_CollapseRoot(t *testing.T) {
	fullTree := genOrUpdateTree(nil, "00", "7", "0000", "8", "0001", "9")
	afterRemove := genOrUpdateTree(fullTree, "00", "")
	requireIsNode(t, afterRemove, "000", "", true, true)
	requireIsNode(t, afterRemove.left, "", "8", false, false)
	requireIsNode(t, afterRemove.right, "", "9", false, false)
}

func TestRemoveValue_OneOfTwoChildren(t *testing.T) {
	fullTree := genOrUpdateTree(nil, "00", "1", "000000", "2", "001111", "3")
	afterRemove := genOrUpdateTree(fullTree, "001111", "")
	requireIsNode(t, afterRemove, "00", "1", true, false)
}

func TestRemoveValue_MissingKey(t *testing.T) {
	baseHash := genOrUpdateTree(nil, "0000", "1", "0011", "1", "0001", "1", "001100", "1")
	hash1 := genOrUpdateTree(baseHash, "0011000101010101", "")
	hash2 := genOrUpdateTree(hash1, "1111", "")
	hash3 := genOrUpdateTree(hash2, "0", "")
	hash4 := genOrUpdateTree(hash3, "0100", "")

	require.EqualValues(t, baseHash.hash, hash1.hash, "tree changed after removing missing key")
	require.EqualValues(t, baseHash.hash, hash2.hash, "tree changed after removing missing key")
	require.EqualValues(t, baseHash.hash, hash3.hash, "tree changed after removing missing key")
	require.EqualValues(t, baseHash.hash, hash4.hash, "tree changed after removing missing key")
}

func TestOrderOfAdditionsDoesNotMatter(t *testing.T) {
	keyValue := []string{"000000", "baz", "0011111", "qux", "000111", "quux1234", "111000", "foo", "1100000", "hello"}
	var1 := []int{2, 6, 0, 8, 4}
	var2 := []int{8, 4, 0, 2, 6}
	var3 := []int{8, 6, 4, 2, 0}

	root1 := genOrUpdateTree(nil, keyValue[var1[0]], keyValue[var1[0]+1], keyValue[var1[1]], keyValue[var1[1]+1],
		keyValue[var1[2]], keyValue[var1[2]+1], keyValue[var1[3]], keyValue[var1[3]+1], keyValue[var1[4]], keyValue[var1[4]+1])

	root2 := genOrUpdateTree(nil, keyValue[var2[0]], keyValue[var2[0]+1], keyValue[var2[1]], keyValue[var2[1]+1],
		keyValue[var2[2]], keyValue[var2[2]+1], keyValue[var2[3]], keyValue[var2[3]+1], keyValue[var2[4]], keyValue[var2[4]+1])

	require.Equal(t, root1.hash, root2.hash, "unexpected different root hash")

	root3 := genOrUpdateTree(nil, keyValue[var3[0]], keyValue[var3[0]+1], keyValue[var3[1]], keyValue[var3[1]+1],
		keyValue[var3[2]], keyValue[var3[2]+1], keyValue[var3[3]], keyValue[var3[3]+1], keyValue[var3[4]], keyValue[var3[4]+1])

	require.Equal(t, root2.hash, root3.hash, "unexpected different root hash")
}

// Tree manipulation
func genOrUpdateTree(root *node, keyValues ...string) *node {
	sandbox := make(dirtyNodes)
	if root == nil {
		root = createEmptyNode()
	}

	for i := 0; i < len(keyValues); i = i + 2 {
		root = insert(hash.CalcSha256([]byte(keyValues[i+1])), nil, 0, root, keyStringToBytes(keyValues[i]), sandbox)
	}

	root = collapseAndHash(root, sandbox, hashAlgoTestNode)
	if root == nil { // special case we got back to empty merkle
		root = createEmptyNode()
	}

	return root
}

func createEmptyNode() *node {
	tmp := createNode([]byte{}, zeroValueHash)
	tmp.hash = hashAlgoTestNode(tmp)
	return tmp
}

func keyStringToBytes(key string) []byte {
	bytesKey := make([]byte, len(key))
	for i, ch := range key {
		bytesKey[i] = char2Byte(uint8(ch))
	}
	return bytesKey
}

func hashAlgoTestNode(n *node) primitives.Sha256 {
	res := make([][]byte, 4)
	res[0] = n.path
	res[1] = n.value
	if n.left != nil {
		res[2] = n.left.hash
	}
	if n.right != nil {
		res[3] = n.right.hash
	}
	return hash.CalcSha256(res...)
}

// required
func requireIsNode(t *testing.T, root *node, key string, value string, hasLeft bool, hasRight bool) {
	require.Equal(t, keyStringToBytes(key), root.path, "wrong key")
	require.Equal(t, hash.CalcSha256([]byte((value))), root.value, "wrong value")
	require.Equal(t, hasLeft, root.left != nil, "wrong left")
	require.Equal(t, hasRight, root.right != nil, "wrong right")
}

func char2Byte(ch uint8) byte {
	if ch == uint8('0') {
		return 0
	}
	return 1
}
