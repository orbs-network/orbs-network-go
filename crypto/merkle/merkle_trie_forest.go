// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package merkle

import (
	"bytes"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"sync"
)

type Forest struct {
	mutex sync.Mutex
	roots []*node
}

func NewForest() (*Forest, primitives.Sha256) {
	var emptyNode = createEmptyTrieNode()
	return &Forest{sync.Mutex{}, []*node{emptyNode}}, emptyNode.hash
}

func createEmptyTrieNode() *node {
	tmp := createNode([]byte{}, zeroValueHash)
	tmp.hash = hashTrieNode(tmp)
	return tmp
}

func (f *Forest) findRoot(rootHash primitives.Sha256) *node {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	for i := len(f.roots) - 1; i >= 0; i-- {
		if f.roots[i].hash.Equal(rootHash) {
			return f.roots[i]
		}
	}

	return nil
}

func (f *Forest) appendRoot(root *node) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.roots = append(f.roots, root)
}

type TrieProofNode struct {
	otherChildHash primitives.Sha256 // the "other child"'s hash
	prefixSize     int               // "my" prefix size
}
type TrieProof struct {
	nodes          []*TrieProofNode
	path           []byte
	extraHashLeft  []byte
	extraHashRight []byte
}

func newTrieProof() *TrieProof {
	return &TrieProof{
		make([]*TrieProofNode, 0, 10), nil, nil, nil,
	}
}

func (tp *TrieProof) appendToProof(n, otherChild *node) {
	tp.nodes = append(tp.nodes, &TrieProofNode{otherChild.hash, len(n.path)})
}

func (f *Forest) GetProof(rootHash primitives.Sha256, path []byte) (*TrieProof, error) {
	current := f.findRoot(rootHash)
	if current == nil {
		return nil, errors.Errorf("unknown root")
	}

	proof := newTrieProof()
	totalPathLen := toBinSize(path)
	currentPathLen := 0
	path = toBin(path, totalPathLen)
	for p := path; bytes.HasPrefix(p, current.path) && len(p) > len(current.path); {
		p = p[len(current.path):]
		currentPathLen += len(current.path)

		parent := current
		sibling := current
		if p[0] == 0 {
			sibling = parent.right
			current = parent.left
		} else {
			sibling = parent.left
			current = parent.right
		}

		if current != nil {
			proof.appendToProof(parent, sibling)
			p = p[1:]
			currentPathLen++
		} else {
			break
		}
	}
	if current != nil { // last node unless wrong key size
		proof.appendToProof(current, current) // last node is on-path-child hash and it's prefix
		// for exclusion: we need to explain how we calculate the last node's hash and where the path diverged
		if !current.isLeaf() {
			proof.extraHashLeft = current.left.hash
			proof.extraHashRight = current.right.hash
		} else {
			proof.extraHashLeft = current.value
		}
		copy(path[currentPathLen:currentPathLen+len(current.path)], current.path) // for exclusion : the path used for proof
	}
	proof.path = path
	return proof, nil
}

func (f *Forest) Verify(rootHash primitives.Sha256, proof *TrieProof, path []byte, valueHash primitives.Sha256) (bool, error) {
	if proof == nil || len(proof.nodes) == 0 {
		return valueHash.Equal(zeroValueHash), nil
	}

	lastNodePathIndex := calculateLastNodePathIndex(proof)
	pathFromVerify := toBin(path, toBinSize(path))

	if !verifyProofIsSelfConsistent(rootHash, proof, pathFromVerify, lastNodePathIndex) {
		return false, errors.Errorf("proof is not self consistent with given key")
	}

	if !valueHash.Equal(zeroValueHash) {
		return verifyProofInclusion(proof, pathFromVerify, valueHash, lastNodePathIndex), nil
	} else {
		return verifyProofExclusion(proof, pathFromVerify, lastNodePathIndex)
	}
}

func calculateLastNodePathIndex(proof *TrieProof) int {
	LastNodeIndex := len(proof.nodes) - 1
	lastNodePathIndex := 0
	for i := 0; i < LastNodeIndex; i++ {
		lastNodePathIndex = lastNodePathIndex + proof.nodes[i].prefixSize + 1
	}
	return lastNodePathIndex
}

func verifyProofIsSelfConsistent(rootHash primitives.Sha256, proof *TrieProof, pathFromVerify []byte, lastNodePathIndex int) bool {
	lastNodeIndex := len(proof.nodes) - 1
	keyEndInd := lastNodePathIndex + proof.nodes[lastNodeIndex].prefixSize
	keyStartInd := lastNodePathIndex
	currentHash := proof.nodes[lastNodeIndex].otherChildHash

	for i := lastNodeIndex - 1; i >= 0; i-- {
		keyEndInd = keyStartInd - 1
		keyStartInd = keyEndInd - proof.nodes[i].prefixSize
		if pathFromVerify[keyEndInd] == 0 {
			currentHash = hashBytes(currentHash, proof.nodes[i].otherChildHash, pathFromVerify[keyStartInd:keyEndInd])
		} else {
			currentHash = hashBytes(proof.nodes[i].otherChildHash, currentHash, pathFromVerify[keyStartInd:keyEndInd])
		}
	}

	return bytes.Equal(currentHash, rootHash)
}

func verifyProofInclusion(proof *TrieProof, verifyPath []byte, verifyValueHash primitives.Sha256, lastNodePathIndex int) bool {
	lastNodeIndex := len(proof.nodes) - 1
	calculatedHash := hashBytes(verifyValueHash, verifyPath[lastNodePathIndex:])
	lastNodeHash := proof.nodes[lastNodeIndex].otherChildHash
	return bytes.Equal(lastNodeHash, calculatedHash)
}

func verifyProofExclusion(proof *TrieProof, pathFromVerify []byte, lastNodePathIndex int) (bool, error) {
	pathLen := len(proof.path)
	if pathLen != len(pathFromVerify) {
		return false, errors.Errorf("proof length is not consistent with given key length")
	}

	lastNodeIndex := len(proof.nodes) - 1
	lastNodePrefix := proof.path[lastNodePathIndex : lastNodePathIndex+proof.nodes[lastNodeIndex].prefixSize]

	var calculatedHash []byte
	if proof.extraHashRight != nil {
		calculatedHash = hashBytes(proof.extraHashLeft, proof.extraHashRight, lastNodePrefix)
	} else {
		calculatedHash = hashBytes(proof.extraHashLeft, lastNodePrefix)
	}
	lastNodeHash := proof.nodes[lastNodeIndex].otherChildHash
	isHashEqual := bytes.Equal(lastNodeHash, calculatedHash)

	isBeginOfPathEqual := bytes.Equal(proof.path[:lastNodePathIndex], pathFromVerify[:lastNodePathIndex])
	isEndOfPathEqual := true
	if lastNodePathIndex < pathLen {
		isEndOfPathEqual = bytes.Equal(lastNodePrefix, pathFromVerify[lastNodePathIndex:])
	}

	return isHashEqual && isBeginOfPathEqual && !isEndOfPathEqual, nil
}

func (f *Forest) Forget(rootHash primitives.Sha256) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.roots[0].hash.Equal(rootHash) { // optimization for most likely use
		f.roots = f.roots[1:]
		return
	}

	found := false
	newRoots := make([]*node, 0, len(f.roots))
	for _, root := range f.roots {
		if found || !root.hash.Equal(rootHash) {
			newRoots = append(newRoots, root)
		} else {
			found = true
		}
	}
	f.roots = newRoots
}

type TrieDiff struct {
	Key   []byte
	Value primitives.Sha256
}
type TrieDiffs []*TrieDiff

func (f *Forest) Update(rootMerkle primitives.Sha256, diffs TrieDiffs) (primitives.Sha256, error) {
	root := f.findRoot(rootMerkle)
	if root == nil {
		return nil, errors.Errorf("must start with valid root")
	}

	sandbox := make(dirtyNodes)

	for _, diff := range diffs {
		root = insert(diff.Value, nil, 0, root, toBin(diff.Key, toBinSize(diff.Key)), sandbox)
	}

	root = collapseAndHash(root, sandbox, hashTrieNode)
	if root == nil { // special case we got back to empty merkle
		root = createEmptyTrieNode()
	}

	f.appendRoot(root)
	return root.hash, nil
}

func hashTrieNode(n *node) primitives.Sha256 {
	if n.isLeaf() {
		return hashBytes(generateLeafParts(n)...)
	} else {
		return hashBytes(generateNodeParts(n)...)
	}
}

func hashBytes(parts ...[]byte) primitives.Sha256 {
	return hash.CalcSha256(parts...)
}

func generateLeafParts(n *node) [][]byte {
	res := make([][]byte, 2)
	res[0] = n.value
	res[1] = n.path
	return res
}

func generateNodeParts(n *node) [][]byte {
	res := make([][]byte, 3)
	res[0] = make([]byte, hash.SHA256_HASH_SIZE_BYTES)
	if n.left != nil {
		copy(res[0], n.left.hash)
	}
	res[1] = make([]byte, hash.SHA256_HASH_SIZE_BYTES)
	if n.right != nil {
		copy(res[1], n.right.hash)
	}
	res[2] = n.path
	return res
}

func toBinSize(s []byte) int {
	return len(s) * 8
}

func toBin(s []byte, size int) []byte {
	bitsArray := make([]byte, size)
	for i := 0; i < size; i++ {
		b := s[i/8]
		bitsArray[i] = 1 & (b >> uint(7-(i%8)))
	}
	return bitsArray
}
