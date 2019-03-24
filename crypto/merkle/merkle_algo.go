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
)

func GetZeroValueHash() primitives.Sha256 {
	return hash.CalcSha256([]byte{})
}

var zeroValueHash = GetZeroValueHash()

type node struct {
	path  []byte
	value primitives.Sha256
	hash  primitives.Sha256
	left  *node
	right *node
}

func createNode(path []byte, valueHash primitives.Sha256) *node {
	return &node{
		path:  path,
		value: valueHash,
		hash:  primitives.Sha256{},
	}
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
	} else {
		n.right = child
	}
}

func (n *node) clone() *node {
	result := &node{
		path:  n.path,
		value: n.value,
		left:  n.left,
		right: n.right,
	}
	return result
}

type dirtyNode struct {
	left, right bool
}
type dirtyNodes map[*node]*dirtyNode

func (dn dirtyNodes) init(node *node) {
	dn[node] = &dirtyNode{}
}

func (dn dirtyNodes) set(node *node, arc byte) {
	if _, exits := dn[node]; !exits {
		dn.init(node)
	}
	if arc == 0 {
		dn[node].left = true
	} else {
		dn[node].right = true
	}
}

func insert(valueHash primitives.Sha256, parent *node, arc byte, current *node, path []byte, sandbox dirtyNodes) *node {
	current = getOrClone(current, parent, arc, sandbox)

	if shouldUpdateCurrent(current, path) {
		current.value = valueHash
		return current
	}

	if shouldUpdateOrCreateChild(current, path) {
		return updateOrCreateChild(current, path, valueHash, sandbox)
	}

	if shouldCreateParent(current, path) {
		return createParent(current, path, valueHash, sandbox)
	}

	return createParentAndSibling(current, path, valueHash, sandbox)
}

func getOrClone(current *node, parent *node, arc byte, sandbox dirtyNodes) *node {
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

func shouldUpdateCurrent(current *node, path []byte) bool {
	return bytes.Equal(current.path, path)
}

func shouldUpdateOrCreateChild(current *node, path []byte) bool {
	return bytes.HasPrefix(path, current.path)
}

func updateOrCreateChild(current *node, path []byte, valueHash primitives.Sha256, sandbox dirtyNodes) *node {
	if !current.hasValue() && current.isLeaf() { // replace it
		current.path = path
		current.value = valueHash
	} else {
		childArc := path[len(current.path)]
		childPath := path[len(current.path)+1:]
		if childNode := current.getChild(childArc); childNode != nil {
			current.setChild(childArc, insert(valueHash, current, childArc, childNode, childPath, sandbox))
		} else if valueHash.Equal(zeroValueHash) {
			// set to empty value cannot create new children, do nothing
		} else {
			newChild := createNode(childPath, valueHash)
			current.setChild(childArc, newChild)
			sandbox.set(current, childArc)
		}
	}
	return current
}

func shouldCreateParent(current *node, path []byte) bool {
	return bytes.HasPrefix(current.path, path)
}

func createParent(current *node, path []byte, valueHash primitives.Sha256, sandbox dirtyNodes) *node {
	childArc := current.path[len(path)]

	newParent := createNode(path, valueHash)
	newParent.setChild(childArc, current)
	sandbox.set(newParent, childArc)

	current.path = current.path[len(path)+1:]
	return newParent
}

func createParentAndSibling(current *node, path []byte, valueHash primitives.Sha256, sandbox dirtyNodes) *node {
	prefixLastIndex := lastCommonPathIndex(current, path)
	newCommonPath := path[:prefixLastIndex]

	newParent := createNode(newCommonPath, zeroValueHash)
	newCurrentArc := current.path[prefixLastIndex]
	newParent.setChild(newCurrentArc, current)
	sandbox.set(newParent, newCurrentArc)

	current.path = current.path[prefixLastIndex+1:]

	newChild := createNode(path[prefixLastIndex+1:], valueHash)
	newChildArc := path[prefixLastIndex]
	newParent.setChild(newChildArc, newChild)
	sandbox.set(newParent, newChildArc)

	return newParent
}

func lastCommonPathIndex(current *node, path []byte) (i int) {
	for i = 0; i < len(current.path) && i < len(path) && current.path[i] == path[i]; i++ {
	}
	return
}

type nodeHasherFunc func(n *node) primitives.Sha256

func collapseAndHash(current *node, sandbox dirtyNodes, f nodeHasherFunc) *node {
	if !current.isLeaf() {
		collapseDirtyChildren(current, sandbox, f)
	}

	if !current.hasValue() {
		if current.isLeaf() { // prune empty leaf node
			return nil
		} else if current.left != nil && current.right == nil {
			current = collapseOnlyChild(current, 0)
		} else if current.left == nil && current.right != nil {
			current = collapseOnlyChild(current, 1)
		}
	}

	current.hash = f(current)
	return current
}

func collapseDirtyChildren(current *node, sandbox dirtyNodes, f nodeHasherFunc) {
	if sandbox[current] != nil {
		if sandbox[current].left {
			current.setChild(0, collapseAndHash(current.left, sandbox, f))
		}
		if sandbox[current].right {
			current.setChild(1, collapseAndHash(current.right, sandbox, f))
		}
	}
}

func collapseOnlyChild(current *node, onlyChild byte) *node {
	child := current.getChild(onlyChild)
	combinedPath := append(current.path, byte(onlyChild))
	combinedPath = append(combinedPath, child.path...)
	current = child.clone()
	current.path = combinedPath
	return current
}
