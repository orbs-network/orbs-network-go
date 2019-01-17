package merkle

import (
	"bytes"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"sync"
)

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

type TrieProofNode struct {
	hashValue  primitives.Sha256 // the sibling's hash
	prefixSize int               // "my" prefix size
}
type TrieProof []*TrieProofNode

func generateTrieProofNode(current, sibling *node) *TrieProofNode {
	return &TrieProofNode{sibling.hash, len(current.path)}
}

func (f *Forest) GetProof(rootHash primitives.Sha256, path []byte) (TrieProof, error) {
	current := f.findRoot(rootHash)
	if current == nil {
		return nil, errors.Errorf("unknown root")
	}

	proof := make(TrieProof, 0, 10)
	other := current
	proof = append(proof, generateTrieProofNode(current, other)) // TODO make proof to struct ?
	path = toBin(path, toBinSize(path))
	for p := path; bytes.HasPrefix(p, current.path); {
		p = p[len(current.path):]

		if len(p) != 0 {
			if p[0] == 0 {
				other = current.right
				current = current.left
			} else {
				other = current.left
				current = current.right
			}

			if current != nil {
				// each proof node has my sibling's hash but my prefix size
				proof = append(proof, generateTrieProofNode(current, other))
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
	if len(proof) == 0 {
		return value.Equal(zeroValueHash), nil
	}

	proofInd := len(proof) - 1
	keyEndInd := toBinSize(path)
	keyStartInd := keyEndInd - proof[proofInd].prefixSize
	path = toBin(path, toBinSize(path))
	current := hashImpl(value, fromBin(path[keyStartInd:keyEndInd]))
	other := proof[proofInd].hashValue

	for i := proofInd - 1; i >= 0; i-- {
		keyEndInd = keyStartInd - 1
		keyStartInd = keyEndInd - proof[i].prefixSize
		if path[keyEndInd] == 0 {
			current = hashImpl(current, other, fromBin(path[keyStartInd:keyEndInd]))
		} else {
			current = hashImpl(other, current, fromBin(path[keyStartInd:keyEndInd]))
		}
		other = proof[i].hashValue
	}

	if !bytes.Equal(other, current) {
		return value.Equal(zeroValueHash), nil
	}
	return /*!value.Equal(zeroValueHash) &&*/ bytes.Equal(current, rootHash), nil
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
		root = insert(diff.Value, nil, 0, root, toBin(diff.Key, toBinSize(diff.Key)), sandbox)
	}

	root = collapseAndHash(root, sandbox, hashTrieNode)
	if root == nil { // special case we got back to empty merkle
		root = createEmptyTrieNode()
	}

	f.appendRoot(root)
	return root.hash, nil
}

func hashTrieNode(n *node) primitives.Sha256 {
	if n.isLeaf() {
		return hashImpl(generateLeafParts(n)...)
	} else {
		return hashImpl(generateNodeParts(n)...)
	}
}

func hashImpl(parts ...[]byte) primitives.Sha256 {
	return hash.CalcSha256(parts...)
}

func generateLeafParts(n *node) [][]byte {
	res := make([][]byte, 2)
	res[0] = n.value
	res[1] = fromBin(n.path)
	return res
}

func generateNodeParts(n *node) [][]byte {
	res := make([][]byte, 3)
	res[0] = make([]byte, hash.SHA256_HASH_SIZE_BYTES)
	if n.left != nil {
		copy(res[0], n.left.hash)
	}
	res[1] = make([]byte, hash.SHA256_HASH_SIZE_BYTES)
	if n.right != nil {
		copy(res[1], n.right.hash)
	}
	res[2] = fromBin(n.path)
	return res
}

func fromBin(s []byte) []byte {
	res := make([]byte, hash.SHA256_HASH_SIZE_BYTES)
	fullbytes := len(s) / 8
	length := fullbytes * 8

	for i := 0; i < length; i = i + 8 {
		res[i/8] = s[i]<<7 |
			s[i+1]<<6 |
			s[i+2]<<5 |
			s[i+3]<<4 |
			s[i+4]<<3 |
			s[i+5]<<2 |
			s[i+6]<<1 |
			s[i+7]
	}
	leftover := len(s) - length
	for i := 0; i < leftover; i++ {
		res[fullbytes] = res[fullbytes] | s[length+i]<<uint(7-i)
	}

	return res
}

func toBinSize(s []byte) int {
	return len(s) * 8
}
func toBin(s []byte, size int) []byte {
	bitsArray := make([]byte, size)
	for i := 0; i < size; i++ {
		b := s[i/8]
		bitsArray[i] = 1 & (b >> uint(7-(i%8)))
	}
	//for i, b := range s {
	//	bitsArray[i*8] = 1 & (b >> 7)
	//	bitsArray[i*8+1] = 1 & (b >> 6)
	//	bitsArray[i*8+2] = 1 & (b >> 5)
	//	bitsArray[i*8+3] = 1 & (b >> 4)
	//	bitsArray[i*8+4] = 1 & (b >> 3)
	//	bitsArray[i*8+5] = 1 & (b >> 2)
	//	bitsArray[i*8+6] = 1 & (b >> 1)
	//	bitsArray[i*8+7] = 1 & b
	//}
	return bitsArray
}
