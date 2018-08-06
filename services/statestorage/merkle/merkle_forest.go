package merkle

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"strings"
)

type RootId uint64
type Proof []*Node

var emptyNode = &Node{}
var emptyNodeHash = emptyNode.hash()
var zeroValueHash = hash.CalcSha256([]byte{})

type Node struct {
	path     string
	value    primitives.Sha256
	hasValue bool
	branches map[byte]primitives.MerkleSha256
}

func (n *Node) hash() primitives.MerkleSha256 {
	return primitives.MerkleSha256(hash.CalcSha256([]byte(fmt.Sprintf("%+v", n))))
}

type Forest struct {
	roots   map[RootId]primitives.MerkleSha256
	nodes   map[string]*Node
	topRoot RootId
}

//TODO do we need func GetTopRoot()?
func (m *Forest) GetRoot(height RootId) (primitives.MerkleSha256, error) {
	return m.roots[height], nil
}

func (m *Forest) addSingleEntry(path string, valueHash primitives.Sha256) RootId {
	//TODO check if needed after general add is implemented
	if m.roots[m.topRoot].Equal(emptyNodeHash) {
		newNode := newNode(path, valueHash)
		sha256s := newNode.hash()
		m.nodes[sha256s.KeyForMap()] = newNode
		m.topRoot++
		m.roots[m.topRoot] = sha256s
		return m.topRoot
	}

	currentRoot := m.nodes[m.roots[m.topRoot].KeyForMap()]
	newRoot := m.add(currentRoot, path, valueHash)
	sha256s := newRoot.hash()
	m.nodes[sha256s.KeyForMap()] = newRoot
	m.topRoot++
	m.roots[m.topRoot] = sha256s
	return m.topRoot

}

func (n *Node) clone() *Node {
	result := &Node{
		path:     n.path,
		value:    n.value, // TODO - copy?
		hasValue: n.hasValue,
		branches: make(map[byte]primitives.MerkleSha256, len(n.branches)),
	}
	for k,v := range n.branches {
		result.branches[k] = v // TODO - copy?
	}
	return result
}

// TODO - do we need to explicitly treat cases where value remains the same? avoid cloning etc...
func (m *Forest) add(currentNode *Node, path string, valueHash primitives.Sha256) (newNode *Node) {
	newNode = currentNode.clone()
	if currentNode.path == path { // existing leaf node updated
		newNode.value = valueHash
		return
	}

	if strings.HasPrefix(path, currentNode.path) {
		branchSelector := path[len(currentNode.path)]
		childPath := path[len(currentNode.path)+1:]
		var newChild *Node
		if branchHash, exists := currentNode.branches[branchSelector]; exists {
			newChild = m.add(m.nodes[branchHash.KeyForMap()], childPath, valueHash)
			// recurse
		} else {
			newChild = &Node{
				path:     childPath,
				value:    valueHash,
				hasValue: true,
				branches: map[byte]primitives.MerkleSha256{},
			}
		}
		childHash := newChild.hash()
		m.nodes[childHash.KeyForMap()] = newChild
		newNode.branches[branchSelector] = childHash
		return
	}

	//// find the first mismatch in path:
	//i := 0
	//for i = 0; i < len(currentNode.path) && i < len(path) && currentNode.path[i] == path[i]; i++ {
	//}
	//
	//if i == len(path) { // n turns into a branch/leaf node, with new child node representing the previously existing path extension
	//}
	//
	//// if we reach here, i < len(n.path). add a new branch at i with two children:
	return
}

func newNode(path string, valueHash primitives.Sha256) *Node {
	return &Node{
		path:     path,
		value:    valueHash,
		hasValue: true,
		branches: map[byte]primitives.MerkleSha256{},
	}
}

func (m *Forest) Update(rootId RootId, diffs []*protocol.ContractStateDiff) {
	for _, diff := range diffs {
		contract := diff.StringContractName()
		for i := diff.StateDiffsIterator(); i.HasNext(); {
			record := i.NextStateDiffs()
			path := contract + record.StringKey()
			m.addSingleEntry(path, hash.CalcSha256([]byte(record.StringValue())))
			//m.nodes[path] = &Node{path: path, value: hash.CalcSha256([]byte(record.StringValue())), hasValue: true}
		}
	}
}

func (m *Forest) updateStringEntries(keyValues ...string) RootId{
	if len(keyValues) % 2 != 0 {
		panic("expected key value pairs")
	}
	for i := 0; i < len(keyValues); i = i+2 {
		m.addSingleEntry(keyValues[i], hash.CalcSha256([]byte(keyValues[i+1])))
	}
	return m.topRoot
}

func (m *Forest) GetProof(rootId RootId, contract string, key string) (Proof, error) {
	fullPath := contract + key
	root := m.roots[rootId]
	currentNode, exists := m.nodes[root.KeyForMap()]
	proof := make(Proof, 0, 10)

	for p := fullPath; exists && strings.HasPrefix(p, currentNode.path); {
		proof = append(proof, currentNode)
		p = p[len(currentNode.path):]

		if len(p) != 0 {
			currentNode, exists = m.nodes[currentNode.branches[p[0]].KeyForMap()]
			p = p[1:]
		} else {
			break
		}
	}
	return proof, nil
}

func (m *Forest) Verify(rootId RootId, proof Proof, contract string, key string, value string) (bool, error) {
	//TODO split the case where we compare against zero value - to simplify determineValueHashByProof
	valueSha256 := hash.CalcSha256([]byte(value))
	hash, err := determineValueHashByProof(proof, contract+key, m.roots[rootId])
	if err != nil {
		return false, err
	}
	return valueSha256.Equal(hash), nil

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
		if node.hasValue { // key is in trie
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
	nextHash, exists := node.branches[path[len(node.path)]]

	if !exists { // key is not in trie
		return zeroValueHash, nil
	}

	// current top node passes validation, proceed to the next node
	return determineValueHashByProof(proof[1:], path[len(node.path)+1:], nextHash)

}

func NewForest() *Forest {
	return &Forest{
		roots: map[RootId]primitives.MerkleSha256{0: emptyNodeHash},
		nodes: map[string]*Node{emptyNodeHash.KeyForMap(): emptyNode},
	}
}
