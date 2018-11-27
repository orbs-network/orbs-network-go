package merkle

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"sync"
)

//const trieRadix = 16

func GetZeroValueHash() primitives.Sha256 {
	return hash.CalcSha256([]byte{})
}

var zeroValueHash = GetZeroValueHash()

type Proof []*ProofNode

// TODO replace proofNode with membuf/proto
type ProofNode struct {
	path     []byte // TODO parity bool?
	value    primitives.Sha256
	left primitives.MerkleSha256
	right primitives.MerkleSha256
}

func (pn *ProofNode) hash() primitives.MerkleSha256 {
	serializedNode := fmt.Sprintf("%+v", pn)
	return primitives.MerkleSha256(hash.CalcSha256([]byte(serializedNode)))
}

type node struct {
	path     []byte // TODO  parity bool
	value    primitives.Sha256
	hash     primitives.MerkleSha256
	left *node
	right *node
	//branches [trieRadix]*node
	//isLeaf   bool
}

func createNode(path []byte, valueHash primitives.Sha256) *node {
	return &node{
		path:     path,
		value:    valueHash,
		hash:     primitives.MerkleSha256{},
	}
}

func createEmptyNode() *node {
	tmp := createNode([]byte{}, zeroValueHash)
	tmp.hash = tmp.serialize().hash()
	return tmp
}

func (n *node) hasValue() bool {
	return !zeroValueHash.Equal(n.value)
}

func (n *node) isLeaf() bool {
	return n.left == nil && n.right == nil
}

func (n *node) getChild(bit byte) *node {
	if bit == 0 {
		return n.left
	}
	return n.right
}

func (n *node) setChild(bit byte, child *node) {
	if bit == 0 {
		n.left = child
	}
	n.right = child
}

func (n *node) serialize() *ProofNode {
	sn := &ProofNode{
		path:     n.path,
		value:    n.value,
		left: primitives.MerkleSha256{},
		right : primitives.MerkleSha256{},
	}
	if n.left != nil {
		sn.left = n.left.hash
	}
	if n.right != nil {
		sn.right = n.right.hash
	}
	return sn
}

func (n *node) clone() *node {
	result := &node{
		path:     n.path,
		value:    n.value,
left : n.left,
		right: n.right,
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
	path = toBin(path)
	current := f.findRoot(rootHash)
	if current == nil {
		return nil, errors.Errorf("unknown root")
	}

	proof := make(Proof, 0, 10)
	proof = append(proof, current.serialize())

	for p := path; bytes.HasPrefix(p, current.path); {
		p = p[len(current.path):]

		if len(p) != 0 {
			if current = current.getChild(p[0]); current != nil {
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
	path = toBin(path)
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

		if path[len(currentNode.path)] == 0 {
			currentHash = currentNode.left
		} else {
			currentHash = currentNode.right
		}
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

func toBin(s []byte) []byte {
	bitsArray := make([]byte, len(s)*8)
	for i, b := range s {
		bitsArray[i*8] = 1 | b >> 7
		bitsArray[i*8+1] = 1 | b >> 6
		bitsArray[i*8+2] = 1 | b >> 5
		bitsArray[i*8+3] = 1 | b >> 4
		bitsArray[i*8+4] = 1 | b >> 3
		bitsArray[i*8+5] = 1 | b >> 2
		bitsArray[i*8+6] = 1 | b >> 1
		bitsArray[i*8+7] = 1 | b
	}
	return bitsArray
}
