package merkle

import (
	"bytes"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"sync"
)

/*
 * TrieProof code
 */
type TrieProofNode [][]byte
type TrieProof []TrieProofNode

func (pn TrieProofNode) hash() primitives.Sha256 {
	return hash.CalcSha256(pn...)
}
func (pn TrieProofNode) path() []byte {
	return pn[0]
}
func (pn TrieProofNode) value() []byte {
	return pn[1]
}
func (pn TrieProofNode) getChild(c byte) []byte {
	if len(pn) == 2 {
		return primitives.Sha256{}
	} else if c == 0 {
		return pn[1]
	} else {
		return pn[2]
	}
}
func (pn TrieProofNode) getKeySize() int {
	return int(pn[1][0])
}

func generateNodeOrLeafProof(n *node) TrieProofNode {
	if n.isLeaf() {
		return generateLeafProof(n)
	} else {
		return generateNodeProof(n)
	}
}

func generateLeafProof(n *node) TrieProofNode {
	res := make(TrieProofNode, 2)
	res[0] = fromBin(n.path)
	res[1] = make([]byte, hash.SHA256_HASH_SIZE_BYTES)
	copy(res[1], n.value)
	res[1][0] = byte(len(n.path))
	return res
}

func generateNodeProof(n *node) TrieProofNode {
	res := make(TrieProofNode, 3)
	prefixSize := byte(len(n.path))
	res[0] = fromBin(n.path)
	res[1] = generateNodeChildHash(n.left, prefixSize)
	res[2] = generateNodeChildHash(n.right, prefixSize)
	return res
}

func generateNodeChildHash(n *node, prefixSize byte) []byte {
	res := make([]byte, hash.SHA256_HASH_SIZE_BYTES)
	if n != nil {
		copy(res, n.hash)
	}
	res[0] = prefixSize
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

// Forest Code
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
	proof = append(proof, generateNodeOrLeafProof(current))

	path = toBin(path, toBinSize(path))
	for p := path; bytes.HasPrefix(p, current.path); {
		p = p[len(current.path):]

		if len(p) != 0 {
			if current = current.getChild(p[0]); current != nil {
				proof = append(proof, generateNodeOrLeafProof(current))
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
	path = toBin(path, toBinSize(path))
	currentHash := rootHash
	emptyMerkleHash := primitives.Sha256{}

	for i, currentNode := range proof {
		calcHash := currentNode.hash()
		//calcHash[hash.SHA256_HASH_SIZE_BYTES-1] = byte(len(n.path))
		if !bytes.Equal(calcHash[1:31], currentHash[1:31]) { // validate current node against expected hash
			return false, errors.Errorf("proof hash mismatch at node %d", i)
		}
		currentPath := toBin(currentNode[0], currentNode.getKeySize())
		if bytes.Equal(path, currentPath) {
			return value.Equal(currentNode.value()), nil
		}
		if len(path) <= len(currentPath) {
			return value.Equal(zeroValueHash), nil
		}
		if !bytes.HasPrefix(path, currentPath) {
			return value.Equal(zeroValueHash), nil
		}

		currentHash = currentNode.getChild(path[len(currentPath)])
		//if path[len(currentPath)] == 0 {
		//	currentHash = currentNode[1]
		//} else if len(currentNode) == 3 {
		//	currentHash = currentNode[2]
		//}
		path = path[len(currentPath)+1:]

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
	return hash.CalcSha256(generateNodeOrLeafProof(n)...)
}

func toBinSize(s []byte) int {
	return len(s) * 8
}
func toBin(s []byte, size int) []byte {
	bitsArray := make([]byte, size)
	for i := 0; i < size; i++ {
		b := s[i/8]
		bitsArray[i] = 1 & (b >> uint(7-(i%8)))
		//bitsArray[i+1] = 1 & (b >> 6)
		//bitsArray[i+2] = 1 & (b >> 5)
		//bitsArray[i+3] = 1 & (b >> 4)
		//bitsArray[i+4] = 1 & (b >> 3)
		//bitsArray[i+5] = 1 & (b >> 2)
		//bitsArray[i+6] = 1 & (b >> 1)
		//bitsArray[i+7] = 1 & b
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
