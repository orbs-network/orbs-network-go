package merkle

import (
	"fmt"
	"strings"

	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

type Proof []*Node

const trieRadix = 256 // base of the merkle trie. TODO change to 16

func getZeroValueHash() primitives.Sha256 {
	return hash.CalcSha256([]byte{})
}

var zeroValueHash = getZeroValueHash()

type Node struct {
	path     string // TODO replace with []byte + parity bool when moving to trieRadix = 16
	value    primitives.Sha256
	branches *[trieRadix]primitives.MerkleSha256
}

func createNode(path string, valueHash primitives.Sha256) *Node {
	return &Node{
		path:     path,
		value:    valueHash,
		branches: &[trieRadix]primitives.MerkleSha256{},
	}
}
func (n *Node) hasValue() bool {
	return !zeroValueHash.Equal(n.value)
}
func (n *Node) hash() primitives.MerkleSha256 {
	// TODO replace this with membuffers serialization. Sprintf should not reach production
	serializedNode := fmt.Sprintf("%+v", n)
	return primitives.MerkleSha256(hash.CalcSha256([]byte(serializedNode)))
}
func (n *Node) clone() *Node {

	result := &Node{
		path:     n.path,
		value:    n.value, // TODO - copy?
		branches: n.branches,
	}
	for k, v := range n.branches {
		result.branches[k] = v // TODO - copy?
	}
	return result
}

func (n *Node) hasChildren() bool {
	for _, v := range n.branches {
		if v != nil {
			return true
		}
	}
	return false
}

func (n *Node) getSingleChildSelector() *byte {
	var singleChildSelector *byte
	for i, v := range n.branches {
		if v != nil {
			if singleChildSelector != nil {
				return nil
			}
			ib := byte(i)
			singleChildSelector = &ib
		}
	}
	return singleChildSelector
}

type Forest struct {
	nodes map[string]*Node
}

// return the merkle trie & the trie root hash for the empty default trie
func NewForest() (*Forest, primitives.MerkleSha256) {
	var emptyNode = createNode("", zeroValueHash)
	var emptyNodeHash = emptyNode.hash()
	return &Forest{nodes: map[string]*Node{emptyNodeHash.KeyForMap(): emptyNode}}, emptyNodeHash
}

func (f *Forest) connectChildToParentAndSaveChild(childNode, parentNode *Node, selector byte) {
	childHash := childNode.hash()
	parentNode.branches[selector] = childHash
	f.nodes[childHash.KeyForMap()] = childNode
}

func (f *Forest) updateSingleEntry(baseHash primitives.MerkleSha256, path string, valueHash primitives.Sha256) primitives.MerkleSha256 {
	baseNode := f.nodes[baseHash.KeyForMap()]
	var newRoot *Node
	if valueHash.Equal(zeroValueHash) {
		newRoot = f.remove(baseNode, path)
	} else {
		newRoot = f.add(baseNode, path, valueHash)
	}
	sha256 := newRoot.hash()
	f.nodes[sha256.KeyForMap()] = newRoot
	return sha256
}

func (f *Forest) squash(n *Node) *Node {
	if !n.hasValue() && !n.hasChildren() && n.path != "" { // no branch and no value - reset path
		clone := n.clone()
		clone.path = ""
		return clone
	}
	if singleChildSelector := n.getSingleChildSelector(); singleChildSelector != nil && !n.hasValue() { // merge with single child
		clone := f.nodes[n.branches[*singleChildSelector].KeyForMap()].clone()
		clone.path = n.path + string([]byte{*singleChildSelector}) + clone.path
		return clone
	}
	return n
}

func (f *Forest) remove(currentNode *Node, path string) *Node {
	if currentNode.path == path { // reached node with value that is being zeroed
		clone := currentNode.clone()
		clone.value = zeroValueHash
		clone = f.squash(clone)
		return clone
	}

	if strings.HasPrefix(path, currentNode.path) {
		clone := currentNode.clone()
		branchSelector := path[len(currentNode.path)]
		branchHash := clone.branches[branchSelector]
		if branchHash != nil {
			newChild := f.remove(f.nodes[branchHash.KeyForMap()], path[len(clone.path)+1:])
			if !newChild.hasChildren() && !newChild.hasValue() {
				clone.branches[branchSelector] = nil
			} else {
				f.connectChildToParentAndSaveChild(newChild, clone, branchSelector)
			}
			clone = f.squash(clone)
		}
		return clone
	}
	return currentNode
}

func (f *Forest) add(currentNode *Node, path string, valueHash primitives.Sha256) *Node {
	newNode := currentNode.clone()
	if currentNode.path == path { // existing leaf node updated
		newNode.value = valueHash
		return newNode
	}

	if strings.HasPrefix(path, currentNode.path) {
		if !currentNode.hasValue() && !currentNode.hasChildren() { // this node has no children and no value, replace it
			newNode = createNode(path, valueHash)
			return newNode
		}

		branchSelector := path[len(currentNode.path)]
		childPath := path[len(currentNode.path)+1:]
		var newChild *Node
		if branchHash := currentNode.branches[branchSelector]; branchHash != nil {
			newChild = f.add(f.nodes[branchHash.KeyForMap()], childPath, valueHash)
		} else {
			newChild = createNode(childPath, valueHash)
		}
		f.connectChildToParentAndSaveChild(newChild, newNode, branchSelector)
		return newNode
	}

	if strings.HasPrefix(currentNode.path, path) { // "insert" a leaf node along the path of currentNode
		branchSelector := newNode.path[len(path)]
		newNode.path = newNode.path[len(path)+1:]
		newParent := createNode(path, valueHash)
		f.connectChildToParentAndSaveChild(newNode, newParent, branchSelector)
		return newParent
	}

	// current node replaced by a new branch node, so that current node is one child and new node is second child
	i := 0
	for i = 0; i < len(currentNode.path) && i < len(path) && currentNode.path[i] == path[i]; i++ {
	}
	newCommonPath := path[:i]
	newParent := createNode(newCommonPath, zeroValueHash)
	newChild := createNode(path[i+1:], valueHash)
	f.connectChildToParentAndSaveChild(newChild, newParent, path[i])

	newNode.path = newNode.path[i+1:]
	f.connectChildToParentAndSaveChild(newNode, newParent, currentNode.path[i])
	return newParent
}

// appends diffs starting at requested trie node (hash) and returns new trie node hash
// NOTE: typical use is baseHash is the newest root hash and return value is new root.
func (f *Forest) Update(baseHash primitives.MerkleSha256, diffs []*protocol.ContractStateDiff) (primitives.MerkleSha256, error) {
	if _, exists := f.nodes[baseHash.KeyForMap()]; !exists {
		return nil, errors.Errorf("root node doesn't exist cannot update trie")
	}

	for _, diff := range diffs {
		contract := diff.StringContractName()
		for i := diff.StateDiffsIterator(); i.HasNext(); {
			record := i.NextStateDiffs()
			path := contract + record.StringKey()
			baseHash = f.updateSingleEntry(baseHash, path, hash.CalcSha256([]byte(record.StringValue())))
		}
	}
	return baseHash, nil
}

// extract and return a verifiable proof for the value of key in the state snapshot reflected by trieId (corresponding to some block height)
func (f *Forest) GetProof(rootHash primitives.MerkleSha256, contract string, key string) (Proof, error) {
	fullPath := contract + key
	currentNode, exists := f.nodes[rootHash.KeyForMap()]
	proof := make(Proof, 0, 10)
	proof = append(proof, currentNode)

	for p := fullPath; exists && strings.HasPrefix(p, currentNode.path); {
		p = p[len(currentNode.path):]

		if p != "" {
			currentNode, exists = f.nodes[currentNode.branches[p[0]].KeyForMap()]
			if exists {
				proof = append(proof, currentNode)
			}
			p = p[1:]
		} else {
			break
		}
	}
	return proof, nil
}

// return true if proof and merkle rootHash validate value for key. false if it confirms value does not match key
// return an error if the proof is inconsistent internally, or, with key, value or rootHash
func (f *Forest) Verify(rootHash primitives.MerkleSha256, proof Proof, contract string, key string, value string) (bool, error) {
	//TODO split the case where we compare against zero value - to simplify determineValueHashByProof
	valueSha256 := hash.CalcSha256([]byte(value))
	expectedHash, err := determineValueHashByProof(proof, contract+key, rootHash)
	if err != nil {
		return false, err
	}
	return valueSha256.Equal(expectedHash), nil
}

func determineValueHashByProof(proof Proof, path string, parentHash primitives.MerkleSha256) (primitives.Sha256, error) {
	if len(proof) == 0 { // proof has ended before a positive conclusion could be reached
		return nil, errors.Errorf("Proof incomplete")
	}

	node := proof[0] // each iteration inspects the top (remaining) node in the proof

	if !node.hash().Equal(parentHash) { // validate current node against expected hash
		return nil, errors.Errorf("Merkle root mismatch or proof may have been tampered with")
	}

	if path == node.path { // current node consumes the remainder of the key. check hasValue value
		if node.hasValue() { // key is in trie
			return node.value, nil
		} else { // key is not in trie
			return zeroValueHash, nil
		}
	} else if len(path) <= len(node.path) { // key is not in trie
		return zeroValueHash, nil
	}

	if !strings.HasPrefix(path, node.path) { // key is not in trie
		return zeroValueHash, nil
	}

	// follow branch: get the hash code of the next expected node for our key
	nextHash := node.branches[path[len(node.path)]]

	if nextHash == nil { // key is not in trie
		return zeroValueHash, nil
	}

	// current top node passes validation, proceed to the next node
	return determineValueHashByProof(proof[1:], path[len(node.path)+1:], nextHash)

}
