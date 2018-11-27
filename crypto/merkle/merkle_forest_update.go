package merkle

import (
	"bytes"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
)

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

type TrieDiff struct {
	Key   []byte
	Value primitives.Sha256
}
type TrieDiffs []*TrieDiff

func (f *Forest) Update(rootMerkle primitives.MerkleSha256, diffs TrieDiffs) (primitives.MerkleSha256, error) {
	root := f.findRoot(rootMerkle)
	if root == nil {
		return nil, errors.Errorf("must start with valid root")
	}

	sandbox := make(dirtyNodes)

	for _, diff := range diffs {
		root = f.insert(diff.Value, nil, 0, root, toBin(diff.Key), sandbox)
	}

	root = f.collapseAndHash(root, sandbox)
	if root == nil { // special case we got back to empty merkle
		root = createEmptyNode()
	}

	f.appendRoot(root)
	return root.hash, nil
}

func (f *Forest) insert(valueHash primitives.Sha256, parent *node, arc byte, current *node, path []byte, sandbox dirtyNodes) *node {
	current = f.getOrClone(current, parent, arc, sandbox)

	if f.shouldUpdateCurrent(current, path) {
		current.value = valueHash
		return current
	}

	if f.shouldUpdateOrCreateChild(current, path) {
		return f.updateOrCreateChild(current, path, valueHash, sandbox)
	}

	if f.shouldCreateParent(current, path) {
		return f.createParent(current, path, valueHash, sandbox)
	}

	return f.createParentAndSibling(current, path, valueHash, sandbox)
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

func (f *Forest) shouldUpdateCurrent(current *node, path []byte) bool {
	return bytes.Equal(current.path, path)
}

func (f *Forest) shouldUpdateOrCreateChild(current *node, path []byte) bool {
	return bytes.HasPrefix(path, current.path)
}

func (f *Forest) updateOrCreateChild(current *node, path []byte, valueHash primitives.Sha256, sandbox dirtyNodes) *node {
	if !current.hasValue() && current.isLeaf() { // replace it
		current.path = path
		current.value = valueHash
	} else {
		childArc := path[len(current.path)]
		childPath := path[len(current.path)+1:]
		if childNode := current.getChild(childArc); childNode != nil {
			current.setChild(childArc, f.insert(valueHash, current, childArc, childNode, childPath, sandbox))
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

func (f *Forest) shouldCreateParent(current *node, path []byte) bool {
	return bytes.HasPrefix(current.path, path)
}

func (f *Forest) createParent(current *node, path []byte, valueHash primitives.Sha256, sandbox dirtyNodes) *node {
	childArc := current.path[len(path)]

	newParent := createNode(path, valueHash)
	newParent.setChild(childArc, current)
	sandbox.set(newParent, childArc)

	current.path = current.path[len(path)+1:]
	return newParent
}

func (f *Forest) createParentAndSibling(current *node, path []byte, valueHash primitives.Sha256, sandbox dirtyNodes) *node {
	prefixLastIndex := f.lastCommonPathIndex(current, path)
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

func (f *Forest) lastCommonPathIndex(current *node, path []byte) (i int) {
	for i = 0; i < len(current.path) && i < len(path) && current.path[i] == path[i]; i++ {
	}
	return
}

func (f *Forest) collapseAndHash(current *node, sandbox dirtyNodes) *node {
	numChildren := 0
	lastOrOnlyChild := byte(0)

	if !current.isLeaf() {
		f.collapseDirtyChildren(current, sandbox)

		numChildren, lastOrOnlyChild = f.countChildrenFindLast(current)
	}

	if !current.hasValue() {
		if current.isLeaf() { // prune empty leaf node
			return nil
		} else if numChildren == 1 {
			current = f.collapseOnlyChild(current, lastOrOnlyChild)
		}
	}

	current.hash = current.serialize().hash()
	return current
}

func (f *Forest) collapseDirtyChildren(current *node, sandbox dirtyNodes) {
	if sandbox[current] != nil {
		if sandbox[current].left {
			current.setChild(0, f.collapseAndHash(current.left, sandbox))
		}
		if sandbox[current].right {
			current.setChild(1, f.collapseAndHash(current.right, sandbox))
		}
	}
}

func (f *Forest) countChildrenFindLast(current *node) (numChildren int, lastOrOnlyChild byte) {
	if current.left != nil {
		numChildren++
		lastOrOnlyChild = 0
	}
	if current.right != nil {
		numChildren++
		lastOrOnlyChild = 1
	}
	return
}

func (f *Forest) collapseOnlyChild(current *node, onlyChild byte) *node {
	child := current.getChild(onlyChild)
	combinedPath := append(current.path, byte(onlyChild))
	combinedPath = append(combinedPath, child.path...)
	current = child.clone()
	current.path = combinedPath
	return current
}
