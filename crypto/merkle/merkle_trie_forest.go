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
	nodes     []*TrieProofNode
	path      []byte
	valueHash []byte
}

func newTrieProof() *TrieProof {
	return &TrieProof{
		make([]*TrieProofNode, 0, 10), nil, nil,
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
	if current != nil { // last node, unless wrong key size should be value(leaf) node "closest" to requested path
		proof.appendToProof(current, current) // use same proof node but data is self hash and self prefix
		if !current.isLeaf() {
			proof.nodes[len(proof.nodes)-1].prefixSize = totalPathLen - currentPathLen
		}
		proof.valueHash = current.value                                           // for exclusion : the value used for proof
		copy(path[currentPathLen:currentPathLen+len(current.path)], current.path) // for exclusion : the path used for proof
	}
	proof.path = path
	return proof, nil
}

func (f *Forest) Verify(rootHash primitives.Sha256, proof *TrieProof, path []byte, valueHash primitives.Sha256) (bool, error) {
	if proof == nil || len(proof.nodes) == 0 {
		return valueHash.Equal(zeroValueHash), nil
	}

	pathFromVerify := toBin(path, toBinSize(path))

	if !verifyProofIsSelfConsistent(rootHash, proof, pathFromVerify) {
		return false, errors.Errorf("proof is not self consistent with given key")
	}

	if !valueHash.Equal(zeroValueHash) { // inclusion
		proofValueNodeIndex := len(proof.nodes) - 1
		calcedHash := hashImpl(valueHash, pathFromVerify[len(pathFromVerify)-proof.nodes[proofValueNodeIndex].prefixSize:])
		return bytes.Equal(proof.nodes[proofValueNodeIndex].otherChildHash, calcedHash), nil
	} else {
		return verifyProofExclusion(proof, pathFromVerify)
	}
}

func verifyProofIsSelfConsistent(rootHash primitives.Sha256, proof *TrieProof, pathFromVerify []byte) bool {
	proofValueNode := len(proof.nodes) - 1
	keyEndInd := len(pathFromVerify)
	keyStartInd := keyEndInd - proof.nodes[proofValueNode].prefixSize
	current := proof.nodes[proofValueNode].otherChildHash

	for i := proofValueNode - 1; i >= 0; i-- {
		keyEndInd = keyStartInd - 1
		keyStartInd = keyEndInd - proof.nodes[i].prefixSize
		if pathFromVerify[keyEndInd] == 0 {
			current = hashImpl(current, proof.nodes[i].otherChildHash, pathFromVerify[keyStartInd:keyEndInd])
		} else {
			current = hashImpl(proof.nodes[i].otherChildHash, current, pathFromVerify[keyStartInd:keyEndInd])
		}
	}

	return bytes.Equal(current, rootHash)
}

func verifyProofExclusion(proof *TrieProof, pathFromVerify []byte) (bool, error) {
	proofValueNodeIndex := len(proof.nodes) - 1
	pathLen := len(proof.path)
	if pathLen != len(pathFromVerify) {
		return false, errors.Errorf("proof length is not consistent with given key length")
	}
	valueNodePrefixIndex := pathLen - proof.nodes[proofValueNodeIndex].prefixSize
	//isHashEqual := bytes.Equal(proof.nodes[proofValueNodeIndex].otherChildHash, hashImpl(proof.valueHash, proof.path[valueNodePrefixIndex:]))
	isBeginOfKeyEqual := bytes.Equal(proof.path[:valueNodePrefixIndex], pathFromVerify[:valueNodePrefixIndex])
	isEndOfKeyEqual := false
	if pathLen-proof.nodes[proofValueNodeIndex].prefixSize != 0 {
		isEndOfKeyEqual = bytes.Equal(proof.path[valueNodePrefixIndex:], pathFromVerify[valueNodePrefixIndex:])
	}
	return /*isHashEqual &&*/ isBeginOfKeyEqual && !isEndOfKeyEqual, nil

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
		return hashImpl(generateLeafParts(n)...)
	} else {
		return hashImpl(generateNodeParts(n)...)
	}
}

func hashImpl(parts ...[]byte) primitives.Sha256 {
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
