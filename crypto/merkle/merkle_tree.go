package merkle

import (
	"bytes"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"math"
)

type TreeProof []primitives.MerkleSha256

type Tree struct {
	root    *node
	keySize int
	maxKey  int
}

func calculateTreeRootCollapseOneLevel(src, dst []primitives.Sha256, size int) int {
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

func CalculateTreeRoot(values []primitives.Sha256) primitives.MerkleSha256 {
	nodes := make([]primitives.Sha256, len(values)/2+1)
	n := calculateTreeRootCollapseOneLevel(values, nodes, len(values))

	iteration := int(math.Ceil(math.Log2(float64(len(values)))))
	for i := 1; i < iteration; i++ {
		n = calculateTreeRootCollapseOneLevel(nodes, nodes, n)
	}
	return primitives.MerkleSha256(nodes[0])
}

func newTree(values []primitives.Sha256) *Tree {
	keySize := int(math.Ceil(math.Log2(float64(len(values)))))
	root := create(values, keySize)
	return &Tree{root, keySize, len(values) - 1}
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
func treeHash(n *node) primitives.MerkleSha256 {
	if n.isLeaf() {
		return primitives.MerkleSha256(n.value)
	}
	return primitives.MerkleSha256(hashTwo(primitives.Sha256(n.left.hash), primitives.Sha256(n.right.hash))) // TODO REMOVE CAST
}

func (t *Tree) GetProof(index int) (TreeProof, error) {
	if index < 0 || index > t.maxKey {
		return nil, errors.Errorf("index for proof is out of bounds")
	}
	proof := make(TreeProof, 0, t.keySize)
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

func (t *Tree) Verify(value primitives.Sha256, proof TreeProof, root primitives.MerkleSha256) error {
	current := value
	for i := len(proof) - 1; i >= 0; i-- {
		current = hashTwo(current, primitives.Sha256(proof[i])) // TODO remove cast
	}

	if !bytes.Equal(root, primitives.MerkleSha256(current)) {
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
