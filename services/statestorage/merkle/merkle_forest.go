package merkle

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"sync"
)

const trieRadix = 16

func GetZeroValueHash() primitives.Sha256 {
	return hash.CalcSha256([]byte{})
}

var zeroValueHash = GetZeroValueHash()

type Proof []*ProofNode

// TODO replace proofNode with membuf/proto
type ProofNode struct {
	path     []byte // TODO parity bool?
	value    primitives.Sha256
	branches [trieRadix]primitives.MerkleSha256
}

func (pn *ProofNode) hash() primitives.MerkleSha256 {
	serializedNode := fmt.Sprintf("%+v", pn)
	return primitives.MerkleSha256(hash.CalcSha256([]byte(serializedNode)))
}

type node struct {
	path     []byte // TODO  parity bool
	value    primitives.Sha256
	hash     primitives.MerkleSha256
	branches [trieRadix]*node
	isLeaf   bool
}

func createNode(path []byte, valueHash primitives.Sha256, isLeaf bool) *node {
	return &node{
		path:     path,
		value:    valueHash,
		branches: [trieRadix]*node{},
		isLeaf:   isLeaf,
		hash:     primitives.MerkleSha256{},
	}
}

func createEmptyNode() *node {
	tmp := createNode([]byte{}, zeroValueHash, true)
	tmp.hash = tmp.serialize().hash()
	return tmp
}

func (n *node) hasValue() bool {
	return !zeroValueHash.Equal(n.value)
}

func (n *node) serialize() *ProofNode {
	sn := &ProofNode{
		path:     n.path,
		value:    n.value,
		branches: [trieRadix]primitives.MerkleSha256{},
	}
	if !n.isLeaf {
		for k, v := range n.branches {
			if v != nil {
				sn.branches[k] = v.hash
			}
		}
	}
	return sn
}

func (n *node) clone() *node {
	newBranches := [trieRadix]*node{}
	if !n.isLeaf {
		copy(newBranches[:], n.branches[:])
	}
	result := &node{
		path:     n.path,
		value:    n.value,
		branches: newBranches,
		isLeaf:   n.isLeaf,
	}
	return result
}

type Forest struct {
	mutex sync.Mutex
	roots []*node
}

func NewForest() (*Forest, primitives.MerkleSha256) {
	var emptyNode = createEmptyNode()
	return &Forest{sync.Mutex{}, []*node{emptyNode}}, emptyNode.hash
}

func (f *Forest) findRoot(rootHash primitives.MerkleSha256) *node {
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

func (f *Forest) GetProof(rootHash primitives.MerkleSha256, path []byte) (Proof, error) {
	path = toHex(path)
	current := f.findRoot(rootHash)
	if current == nil {
		return nil, errors.Errorf("unknown root")
	}

	proof := make(Proof, 0, 10)
	proof = append(proof, current.serialize())

	for p := path; bytes.HasPrefix(p, current.path); {
		p = p[len(current.path):]

		if len(p) != 0 {
			if current = current.branches[p[0]]; current != nil {
				proof = append(proof, current.serialize())
				p = p[1:]
			} else {
				break
			}
		} else {
			break
		}
	}
	return proof, nil
}

func (f *Forest) Verify(rootHash primitives.MerkleSha256, proof Proof, path []byte, value primitives.Sha256) (bool, error) {
	path = toHex(path)
	currentHash := rootHash
	emptyMerkleHash := primitives.MerkleSha256{}

	for i, currentNode := range proof {
		calcHash := currentNode.hash()
		if !calcHash.Equal(currentHash) { // validate current node against expected hash
			return false, errors.Errorf("proof hash mismatch at node %d", i)
		}
		if bytes.Equal(path, currentNode.path) {
			return value.Equal(currentNode.value), nil
		}
		if len(path) <= len(currentNode.path) {
			return value.Equal(zeroValueHash), nil
		}
		if !bytes.HasPrefix(path, currentNode.path) {
			return value.Equal(zeroValueHash), nil
		}
		currentHash = currentNode.branches[path[len(currentNode.path)]]
		path = path[len(currentNode.path)+1:]

		if emptyMerkleHash.Equal(currentHash) {
			return value.Equal(zeroValueHash), nil
		}
	}

	return false, errors.Errorf("proof incomplete ")
}

func (f *Forest) Forget(rootHash primitives.MerkleSha256) {
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

func toHex(s []byte) []byte {
	hexBytes := make([]byte, len(s)*2)
	for i, b := range s {
		hexBytes[i*2] = 0xf & (b >> 4)
		hexBytes[i*2+1] = 0x0f & b
	}
	return hexBytes
}
