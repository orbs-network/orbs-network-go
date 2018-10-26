package merkle

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"strings"
	"sync"
)

const trieRadix = 256 // base of the merkle trie. TODO change to 16

func GetZeroValueHash() primitives.Sha256 {
	return hash.CalcSha256([]byte{})
}

var zeroValueHash = GetZeroValueHash()

type Proof []*ProofNode

// TODO replace proofNode with membuf/proto
type ProofNode struct {
	path     string // TODO replace with []byte + parity bool when moving to trieRadix = 16
	value    primitives.Sha256
	branches [trieRadix]primitives.MerkleSha256
}

func (pn *ProofNode) hash() primitives.MerkleSha256 {
	serializedNode := fmt.Sprintf("%+v", pn)
	return primitives.MerkleSha256(hash.CalcSha256([]byte(serializedNode)))
}

type node struct {
	path     string // TODO replace with []byte + parity bool when moving to trieRadix = 16
	value    primitives.Sha256
	hash     primitives.MerkleSha256
	branches [trieRadix]*node
	isLeaf   bool
}

func createNode(path string, valueHash primitives.Sha256, isLeaf bool) *node {
	return &node{
		path:     path,
		value:    valueHash,
		branches: [trieRadix]*node{},
		isLeaf:   isLeaf,
		hash:     primitives.MerkleSha256{},
	}
}

func createEmptyNode() *node {
	tmp := createNode("", zeroValueHash, true)
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
		value:    n.value, // TODO - copy?
		branches: newBranches,
		isLeaf:   n.isLeaf,
	}
	return result
}

type dirtyNodes map[*node]map[byte]bool

func (dn dirtyNodes) init(node *node) {
	dn[node] = make(map[byte]bool)
}

func (dn dirtyNodes) set(node *node, arc byte) {
	if _, exits := dn[node]; !exits {
		dn.init(node)
	}
	dn[node][arc] = true
}

type MerkleDiffs map[string]primitives.Sha256

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

func (f *Forest) GetProof(rootHash primitives.MerkleSha256, path string) (Proof, error) {
	current := f.findRoot(rootHash)
	if current == nil {
		return nil, errors.Errorf("unknown root")
	}

	proof := make(Proof, 0, 10)
	proof = append(proof, current.serialize())

	for p := path; strings.HasPrefix(p, current.path); {
		p = p[len(current.path):]

		if p != "" {
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

func (f *Forest) Verify(rootHash primitives.MerkleSha256, proof Proof, path string, value primitives.Sha256) (bool, error) {
	currentHash := rootHash
	emptyMerkleHash := primitives.MerkleSha256{}

	for i, currentNode := range proof {
		calcHash := currentNode.hash()
		if !calcHash.Equal(currentHash) { // validate current node against expected hash
			return false, errors.Errorf("proof hash mismatch at node %d", i)
		}
		if path == currentNode.path {
			return value.Equal(currentNode.value), nil
		}
		if len(path) <= len(currentNode.path) {
			return value.Equal(zeroValueHash), nil
		}
		if !strings.HasPrefix(path, currentNode.path) {
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

type GcFunc func(root primitives.MerkleSha256)

func (f *Forest) GcFunc() GcFunc {
	return func(root primitives.MerkleSha256) {
		f.forget(root)
	}
}

func (f *Forest) forget(rootHash primitives.MerkleSha256) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	newRoots := make([]*node, 0, len(f.roots)-1)
	for _, root := range f.roots { // naive copy because its a small array and simple code is better
		if !root.hash.Equal(rootHash) {
			newRoots = append(newRoots, root)
		}
	}
	f.roots = newRoots
}

func (f *Forest) Update(rootMerkle primitives.MerkleSha256, diffs MerkleDiffs) (primitives.MerkleSha256, error) {
	root := f.findRoot(rootMerkle)
	if root == nil {
		return nil, errors.Errorf("must start with valid root")
	}

	sandbox := make(dirtyNodes)

	for path, value := range diffs {
		root = f.travelUpdateAndMark(nil, 0, root, path, value, sandbox)
	}

	root = f.travelCollapseAndHash(root, sandbox)
	if root == nil { // special case we got back to empty merkle
		root = createEmptyNode()
	}

	f.appendRoot(root)
	return root.hash, nil
}

func (f *Forest) travelUpdateAndMark(parent *node, arc byte, current *node, path string, valueHash primitives.Sha256, sandbox dirtyNodes) *node {
	current = f.getOrClone(current, parent, arc, sandbox)

	if current.path == path { // path reached exactly
		current.value = valueHash
		return current
	}

	if strings.HasPrefix(path, current.path) { // current is next part of path
		if !current.hasValue() && current.isLeaf { // replace it
			current.path = path
			current.value = valueHash
		} else {
			childArc := path[len(current.path)]
			childPath := path[len(current.path)+1:]
			if childNode := current.branches[childArc]; childNode != nil {
				current.branches[childArc] = f.travelUpdateAndMark(current, childArc, childNode, childPath, valueHash, sandbox)
			} else if valueHash.Equal(zeroValueHash) {
				// set to empty value cannot create new children, do nothing
			} else {
				newChild := createNode(childPath, valueHash, true)
				current.branches[childArc] = newChild
				current.isLeaf = false
				sandbox.set(current, childArc)
			}
		}
		return current
	}

	if strings.HasPrefix(current.path, path) { // "insert" a valued node along the path
		childArc := current.path[len(path)]

		newParent := createNode(path, valueHash, false)
		newParent.branches[childArc] = current
		sandbox.set(newParent, childArc)

		current.path = current.path[len(path)+1:]
		return newParent
	}

	// new node is a brother of mine so i create a common parent too
	i := 0
	for i = 0; i < len(current.path) && i < len(path) && current.path[i] == path[i]; i++ {
	}
	newCommonPath := path[:i]

	newParent := createNode(newCommonPath, zeroValueHash, false)
	newParent.branches[current.path[i]] = current
	sandbox.set(newParent, current.path[i])

	current.path = current.path[i+1:]

	newChild := createNode(path[i+1:], valueHash, true)
	newParent.branches[path[i]] = newChild
	sandbox.set(newParent, path[i])

	return newParent
}

func (f *Forest) getOrClone(current *node, parent *node, arc byte, sandbox dirtyNodes) *node {
	var actual *node
	if exists := sandbox[current]; exists != nil {
		actual = current
	} else {
		actual = current.clone()
		sandbox.init(actual)
		sandbox.set(parent, arc)
	}
	return actual
}

func (f *Forest) travelCollapseAndHash(current *node, sandbox dirtyNodes) *node {
	nChildren := 0
	aChild := 0

	if !current.isLeaf {
		for arc := range sandbox[current] {
			current.branches[arc] = f.travelCollapseAndHash(current.branches[arc], sandbox)
		}

		// check if i have any children left: count+save last one
		for arc, child := range current.branches {
			if child != nil {
				nChildren++
				aChild = arc
			}
		}
		current.isLeaf = nChildren == 0
	}

	// if i have no value ...
	if !current.hasValue() {
		if current.isLeaf { // prune empty leaf node
			return nil
		} else if nChildren == 1 { // fold up only child
			child := current.branches[aChild]
			combinedPath := current.path + string([]byte{byte(aChild)}) + child.path
			current = child.clone()
			current.path = combinedPath
		}
	}

	current.hash = current.serialize().hash()
	return current
}
