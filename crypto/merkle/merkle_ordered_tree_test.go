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
	"strconv"
	"testing"
	"time"
)

func TestTreeNodeHash(t *testing.T) {
	// Note in our tree implementation there can be nodes that have 0 or 2 children only
	value := hash.CalcSha256([]byte("value sha"))
	left := hash.CalcSha256([]byte("value left"))
	right := hash.CalcSha256([]byte("value right"))

	leaf := &node{
		path:  nil,
		value: value,
		hash:  nil,
		left:  nil,
		right: nil,
	}
	require.Equal(t, treeHash(leaf), value, "leaf node hash mishmatch")

	fullNode := &node{
		path:  nil,
		value: value,
		hash:  nil,
		left: &node{
			path:  nil,
			value: nil,
			hash:  left,
			left:  nil,
			right: nil,
		},
		right: &node{
			path:  nil,
			value: nil,
			hash:  right,
			left:  nil,
			right: nil,
		},
	}
	require.Equal(t, treeHash(fullNode), hashTwoInTest(left, right), "node with only left hash mishmatch")
}

func TestTreeHashAndStructure(t *testing.T) {
	tests := []struct {
		name    string
		values  []int
		keysize int
		maxkey  int
	}{
		{"7 Values", []int{10, 100, -3, 5, 19, 4, 9}, 3, 6},
		{"8 Values", []int{7, 4, 6, -5, 6, 66, 669, -100}, 3, 7},
		{"9 Values", []int{77, 345, -333, 187, 666, 777, 19, 6, -2}, 4, 8},
		{"10 Values", []int{5, -88, 55, 4, 1, 0, 0, 75, -1, -9}, 4, 9},
		{"11 Values", []int{8, 7, 6, 5, 4, 3, 2, 1, 0, -1, -2}, 4, 10},
		{"12 Values", []int{8, 7, 6, 5, 4, 3, 2, 1, 0, -1, -2, 1000}, 4, 11},
		{"13 Values", []int{8, 7, 6, 5, 4, 3, 2, 1, 0, -1, -2, 1000, 55}, 4, 12},
		{"14 Values", []int{8, 7, 6, 5, 4, 3, 2, 1, 0, -1, -2, 1000, 55, 66}, 4, 13},
		{"17 Values", []int{8, 7, 6, 5, 4, 3, 2, 1, 0, -1, -2, 1000, 3, 4, 5, 6, 7}, 5, 16},
	}
	for i := range tests {
		cTest := tests[i] // this is so that we can run tests in parallel, see https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721
		t.Run(cTest.name, func(t *testing.T) {
			t.Parallel()
			hashValues := generateHashValueList(cTest.values)
			tree := NewOrderedTree(hashValues)
			require.Equal(t, hashTreeInTest(tree.root), tree.root.hash, "%s hash mismatch", cTest.name)
			require.Equal(t, cTest.keysize, tree.keySize, "tree max depth size error", cTest.name)
			require.Equal(t, cTest.maxkey, tree.maxKey, "max index is wrong", cTest.name)
		})
	}
}

func TestProofOutOfBounds(t *testing.T) {
	values := []int{0, 1, 2, 3, 4, 5, 6, 7, 8}
	tree := NewOrderedTree(generateHashValueList(values))

	proof, err := tree.GetProof(-5)
	require.Nil(t, proof, "proof should not exist")
	require.Error(t, err, "error should have occurred")

	proof, err = tree.GetProof(len(values))
	require.Nil(t, proof, "proof should not exist")
	require.Error(t, err, "error should have occurred")
}

func TestGetProofInIncomleteTreeShortBranch(t *testing.T) {
	values := []int{7, 4, 6, -5, 6, 66, 669, -100, 5}
	tree := NewOrderedTree(generateHashValueList(values))

	proof, err := tree.GetProof(8)
	require.NotNil(t, proof, "proof should exist")
	require.NoError(t, err, "error should not have occurred")
	require.Equal(t, 1, len(proof), "length of proof wrong")
	require.Equal(t, tree.root.left.hash, proof[0], "proof[0] value wrong")
	require.Equal(t, hashTwoInTest(proof[0], generateHashValue(5)), tree.root.hash, "proof and root don't match")
}

func TestGetProofInCompleteTree(t *testing.T) {
	values := []int{7, 4, 6, -5, 6, 66, 669, -100, 5, 4, -77, -91, 12, 77, 7, 16} // must be list of 2*n
	tree := NewOrderedTree(generateHashValueList(values))

	for i := range values {
		proof, err := tree.GetProof(i)
		require.NotNil(t, proof, "proof should exist for %d", i)
		require.NoError(t, err, "error should not have occurred for %d", i)

		current := tree.root
		for j := tree.keySize - 1; j >= 0; j-- {
			var stepHash primitives.Sha256
			if ((1 << uint(j)) & i) == 0 {
				stepHash = current.right.hash
				current = current.left
			} else {
				stepHash = current.left.hash
				current = current.right
			}
			require.Equal(t, stepHash, proof[tree.keySize-1-j], "proof of %d not match at step %d", i, j)
		}
	}
}

func TestVerifyOnlyCorrectInputWorks(t *testing.T) {
	values := []int{7, 4, 16, -5, 6, 66, 669, -100, 5}
	tree := NewOrderedTree(generateHashValueList(values))
	proof, _ := tree.GetProof(2)

	for i := 0; i < len(values); i++ {
		err := Verify(generateHashValue(values[i]), proof, tree.root.hash)
		if i != 2 {
			require.Error(t, err, "verify should have error at index %d", i)
		} else {
			require.NoError(t, err, "verify should not error at index %d", i)
		}
	}
}

func TestTreeVerifyProofs(t *testing.T) {
	tests := []struct {
		name   string
		values []int
	}{
		{"7 Values", []int{10, 100, -3, 5, 19, 4, 9}},
		{"9 Values", []int{77, 345, -333, 187, 666, 777, 19, 6, -2}},
		{"12 Values", []int{8, 7, 6, 5, 4, 3, 2, 1, 0, -1, -2, 1000}},
		{"13 Values", []int{8, 7, 6, 5, 4, 3, 2, 1, 0, -1, -2, 1000, 55}},
		{"14 Values", []int{8, 7, 6, 5, 4, 3, 2, 1, 0, -1, -2, 1000, 55, 66}},
		{"17 Values", []int{8, 7, 6, 5, 4, 3, 2, 1, 0, -1, -2, 1000, 3, 4, 5, 6, 7}},
	}
	for i := range tests {
		cTest := tests[i] // this is so that we can run tests in parallel, see https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721
		t.Run(cTest.name, func(t *testing.T) {
			t.Parallel()
			tree := NewOrderedTree(generateHashValueList(cTest.values))

			index := 0
			proof1, _ := tree.GetProof(index)
			err := Verify(generateHashValue(cTest.values[index]), proof1, tree.root.hash)
			require.NoError(t, err, "checking index 0")

			index = len(cTest.values) - 1
			proof2, _ := tree.GetProof(index)
			err = Verify(generateHashValue(cTest.values[index]), proof2, tree.root.hash)
			require.NoError(t, err, "checking last index")

			index = len(cTest.values) / 2
			proof3, _ := tree.GetProof(index)
			err = Verify(generateHashValue(cTest.values[index]), proof3, tree.root.hash)
			require.NoError(t, err, "checking middle")
		})
	}
}

func TestTreeCalculatedHash(t *testing.T) {
	tests := []struct {
		name   string
		values []int
	}{
		{"4 Values", []int{0, 1, 2, 3}},
		{"5 Values", []int{0, 1, 2, 3, 4}},
		{"7 Values", []int{10, 100, -3, 5, 19, 4, 9}},
		{"9 Values", []int{77, 345, -333, 187, 666, 777, 19, 6, -2}},
		{"12 Values", []int{8, 7, 6, 5, 4, 3, 2, 1, 0, -1, -2, 1000}},
		{"13 Values", []int{8, 7, 6, 5, 4, 3, 2, 1, 0, -1, -2, 1000, 55}},
		{"14 Values", []int{8, 7, 6, 5, 4, 3, 2, 1, 0, -1, -2, 1000, 55, 66}},
		{"15 Values", []int{8, 7, 6, 5, 4, 3, 2, 1, 0, -1, -2, 1000, 55, 66, 77}},
		{"17 Values", []int{8, 7, 6, 5, 4, 3, 2, 1, 0, -1, -2, 1000, 3, 4, 5, 6, 7}},
	}
	for i := range tests {
		cTest := tests[i] // this is so that we can run tests in parallel, see https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721
		t.Run(cTest.name, func(t *testing.T) {
			hashValues := generateHashValueList(cTest.values)
			tree := NewOrderedTree(hashValues)
			rootCalc := CalculateOrderedTreeRoot(hashValues)
			require.Equal(t, tree.root.hash, rootCalc, "%s calculated hash mismatch", cTest.name)
		})
	}
}

func TestTreeOneValue(t *testing.T) {
	vals := fakeHashValues([]int{1})
	tree := NewOrderedTree(vals)
	require.Equal(t, tree.root.hash, vals[0], "calculated hash mismatch")
	proof, _ := tree.GetProof(0)
	require.Len(t, proof, 0, "wrong length")
	err := Verify(vals[0], proof, tree.root.hash)
	require.NoError(t, err, "proof verification failed")
}

func TestTreeNoValues(t *testing.T) {
	tree := NewOrderedTree(nil)
	require.Equal(t, tree.root.hash, getEmptyHash(), "calculated hash mismatch")
	_, err := tree.GetProof(0)
	require.Error(t, err, "proof cannot be created")
}

// =================
// helpers
// =================

func hashTwoInTest(l, r []byte) primitives.Sha256 {
	s, b := l, r
	for i := range l {
		if l[i] < r[i] {
			break
		}
		if l[i] > r[i] {
			s = r
			b = l
			break
		}
	}
	res := make([]byte, len(s)+len(b))
	for i := 0; i < len(s); i++ {
		res[i] = s[i]
	}
	for i := 0; i < len(b); i++ {
		res[i+len(s)] = b[i]
	}
	return hash.CalcSha256(res)
}

func hashTreeInTest(n *node) primitives.Sha256 {
	if n == nil {
		return zeroValueHash
	}
	if n.isLeaf() {
		return n.value
	}
	return hashTwoInTest(hashTreeInTest(n.left), hashTreeInTest(n.right))
}

func fakeHashValues(vals []int) []primitives.Sha256 {
	hashValues := make([]primitives.Sha256, len(vals))
	for i, v := range vals {
		b := getEmptyHash()
		b[31] = byte(v)
		hashValues[i] = b
	}
	return hashValues
}

func generateHashValue(v int) primitives.Sha256 {
	return hash.CalcSha256([]byte(strconv.Itoa(v)))
}

func generateHashValueList(vals []int) []primitives.Sha256 {
	hashValues := make([]primitives.Sha256, len(vals))
	for i, v := range vals {
		hashValues[i] = generateHashValue(v)
	}
	return hashValues
}

/*
* test to compare creating a tree vs just calculating root directly
 */
func TestTreeStress(t *testing.T) {
	t.SkipNow()
	times := 10000
	nVals := 1000

	values := make([]int, nVals)
	for i := 0; i < nVals; i++ {
		values[i] = i
	}
	hashValues := fakeHashValues(values)

	start := time.Now()
	for i := 0; i < times; i++ {
		NewOrderedTree(hashValues)
	}
	duration := time.Now().Sub(start)
	t.Logf("created merkle trees in %v", duration)

	start = time.Now()
	for i := 0; i < times; i++ {
		CalculateOrderedTreeRoot(hashValues)
	}
	duration = time.Now().Sub(start)
	t.Logf("calculated merkle trees in %v", duration)
}
