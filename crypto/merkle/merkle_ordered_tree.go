package merkle

import (
	"bytes"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"math"
)

type OrderedTreeProof []primitives.Sha256

type OrderedTree struct {
	root    *node
	keySize int
	maxKey  int
}

func calculateOrderedTreeRootCollapseOneLevel(src, dst []primitives.Sha256, size int) int {
	for j := 1; j < size; j = j + 2 {
		dst[j/2] = hashTwo(src[j-1], src[j])
	}
	if size%2 != 0 {
		dst[size/2] = src[size-1]
		size = size/2 + 1
	} else {
		size = size / 2
	}
	return size
}

func CalculateOrderedTreeRoot(values []primitives.Sha256) primitives.Sha256 {
	nodes := make([]primitives.Sha256, len(values)/2+1)
	n := calculateOrderedTreeRootCollapseOneLevel(values, nodes, len(values))

	iteration := int(math.Ceil(math.Log2(float64(len(values)))))
	for i := 1; i < iteration; i++ {
		n = calculateOrderedTreeRootCollapseOneLevel(nodes, nodes, n)
	}
	return nodes[0]
}

func NewOrderedTree(values []primitives.Sha256) *OrderedTree {
	keySize := int(math.Ceil(math.Log2(float64(len(values)))))
	root := create(values, keySize)
	return &OrderedTree{root, keySize, len(values) - 1}
}

func create(values []primitives.Sha256, keySize int) *node {
	root := &node{}
	sandbox := make(dirtyNodes)

	for i, value := range values {
		root = insert(value, nil, 0, root, toKey(i, keySize), sandbox)
	}

	root = collapseAndHash(root, sandbox, treeHash)
	return root
}

// NOTE : practical - we don't have a node with just one child.
func treeHash(n *node) primitives.Sha256 {
	if n.isLeaf() {
		return n.value
	}
	return hashTwo(n.left.hash, n.right.hash)
}

func (t *OrderedTree) GetProof(index int) (OrderedTreeProof, error) {
	if index < 0 || index > t.maxKey {
		return nil, errors.Errorf("index for proof is out of bounds")
	}
	proof := make(OrderedTreeProof, 0, t.keySize)
	keyInBytes := toKey(index, t.keySize)
	current := t.root
	other := t.root
	for i := 0; i < t.keySize; i++ {
		i = i + len(current.path) // skip any residual prefix
		if keyInBytes[i] == 0 {
			other = current.right
			current = current.left
		} else {
			other = current.left
			current = current.right
		}

		proof = append(proof, other.hash)

		if current.isLeaf() {
			break
		}
	}

	return proof, nil
}

func Verify(value primitives.Sha256, proof OrderedTreeProof, root primitives.Sha256) error {
	current := value
	for i := len(proof) - 1; i >= 0; i-- {
		current = hashTwo(current, proof[i])
	}

	if !bytes.Equal(root, current) {
		return errors.Errorf("proof hash did not match the root")
	}
	return nil
}

func orderHashes(left, right primitives.Sha256) (small, big primitives.Sha256) {
	small, big = left, right
	for i := range left {
		if left[i] < right[i] {
			break
		}
		if left[i] > right[i] {
			small = right
			big = left
			break
		}
	}
	return
}

func hashTwo(left, right primitives.Sha256) primitives.Sha256 {
	small, big := orderHashes(left, right)
	result := append(small, big...)
	return hash.CalcSha256(result)
}

func toKey(index int, keySize int) []byte {
	key := make([]byte, keySize)
	for i := keySize - 1; i >= 0; i-- {
		key[i] = byte(index & 1)
		index = index >> 1
	}
	return key
}
