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

type TrieProof []*TrieProofNode

// TODO replace proofNode with membuf/proto
type TrieProofNode struct {
	path  []byte // TODO parity bool?
	value primitives.Sha256
	left  primitives.Sha256
	right primitives.Sha256
}

func newProofNode(n* node) *TrieProofNode {
	pn := &TrieProofNode{
		path:  n.path,
		value: n.value,
		left:  primitives.Sha256{},
		right: primitives.Sha256{},
	}
	if n.left != nil {
		pn.left = n.left.hash
	}
	if n.right != nil {
		pn.right = n.right.hash
	}
	return pn
}

func (pn *TrieProofNode) hash() primitives.Sha256 {
	serializedNode := fmt.Sprintf("%+v", pn)
	return hash.CalcSha256([]byte(serializedNode))
}

func createEmptyTrieNode() *node {
	tmp := createNode([]byte{}, zeroValueHash)
	tmp.hash = hashTrieNode(tmp)
	return tmp
}

type Forest struct {
	mutex sync.Mutex
	roots []*node
}

func NewForest() (*Forest, primitives.Sha256) {
	var emptyNode = createEmptyTrieNode()
	return &Forest{sync.Mutex{}, []*node{emptyNode}}, emptyNode.hash
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

func (f *Forest) GetProof(rootHash primitives.Sha256, path []byte) (TrieProof, error) {
	current := f.findRoot(rootHash)
	if current == nil {
		return nil, errors.Errorf("unknown root")
	}

	proof := make(TrieProof, 0, 10)
	proof = append(proof, newProofNode(current))

	path = toBin(path)
	for p := path; bytes.HasPrefix(p, current.path); {
		p = p[len(current.path):]

		if len(p) != 0 {
			if current = current.getChild(p[0]); current != nil {
				proof = append(proof, newProofNode(current))
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

func (f *Forest) Verify(rootHash primitives.Sha256, proof TrieProof, path []byte, value primitives.Sha256) (bool, error) {
	path = toBin(path)
	currentHash := rootHash
	emptyMerkleHash := primitives.Sha256{}

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
		root = insert(diff.Value, nil, 0, root, toBin(diff.Key), sandbox)
	}

	root = collapseAndHash(root, sandbox, hashTrieNode)
	if root == nil { // special case we got back to empty merkle
		root = createEmptyTrieNode()
	}

	f.appendRoot(root)
	return root.hash, nil
}

func hashTrieNode(n *node) primitives.Sha256 {
	return newProofNode(n).hash()
}

func toBin(s []byte) []byte {
	bitsArray := make([]byte, len(s)*8)
	for i, b := range s {
		bitsArray[i*8] = 1 & (b >> 7)
		bitsArray[i*8+1] = 1 & (b >> 6)
		bitsArray[i*8+2] = 1 & (b >> 5)
		bitsArray[i*8+3] = 1 & (b >> 4)
		bitsArray[i*8+4] = 1 & (b >> 3)
		bitsArray[i*8+5] = 1 & (b >> 2)
		bitsArray[i*8+6] = 1 & (b >> 1)
		bitsArray[i*8+7] = 1 & b
	}
	return bitsArray
}
