package merkle

import (
	"encoding/base64"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

//TODO - updateStringEntries should advance TrieId only by one

//TODO - updateStringEntries - the bulk update version (optimize node access)
//TODO - Work with persistent state adapter + cache where appropriate

//TODO - serialization based on spec (oded)
//TODO - Radix 16 +/- parity
//TODO - split branch and node leafs (this can be limited to serialization only)
//TODO - avoid hashing values of less than 32 bytes ?? Other optimizations (see ethereum)?
//TODO - what hash functions should be used for values and what functions for node addresses?
//TODO - should we include full values or just hashes (compare Ethereum)
//TODO - use hashes of contract names

//TODO - garbage collection
//TODO - in case uniform key length is enforced - accept a key length in the forest constructor
//TODO - getProof in bulk ???

type forest struct {
	forest *Forest
	roots map[primitives.BlockHeight]primitives.MerkleSha256
	top primitives.BlockHeight
}

func newForest() *forest {
	return &forest{
		forest: NewForest(),
		roots: map[primitives.BlockHeight]primitives.MerkleSha256{0: GetEmptyNodeHash()},
		top: 0,
	}
}

func (f *forest) getRootHash (id primitives.BlockHeight) (primitives.MerkleSha256, error) {
	return f.roots[id], nil
}

func (f *forest) getTopRootHash () (primitives.MerkleSha256, error) {
	return f.roots[f.top], nil
}

func (f *forest) Update(baseHash primitives.MerkleSha256, diffs []*protocol.ContractStateDiff) primitives.MerkleSha256 {
	newRoot := f.forest.Update(baseHash, diffs)
	f.top++
	f.roots[f.top] = newRoot
	return newRoot
}

func updateStringEntries(f *forest, keyValues ...string) primitives.BlockHeight {
	if len(keyValues)%2 != 0 {
		panic("expected key value pairs")
	}
	var root primitives.MerkleSha256
	for i := 0; i < len(keyValues); i = i + 2 {
		root = f.forest.updateSingleEntry(f.roots[f.top], keyValues[i], hash.CalcSha256([]byte(keyValues[i+1])))
		f.top++
		f.roots[f.top] =  root
	}
	return f.top
}

func verifyProof(t *testing.T, f *forest, rootId primitives.BlockHeight, proof Proof, contract string, key string, value string, exists bool) {
	rootHash, _ := f.getRootHash(rootId)
	f.dump(t)
	verified, err := f.forest.Verify(rootHash, proof, contract, key, value)
	require.NoError(t, err, "proof verification failed")
	require.Equal(t, exists, verified, "proof verification returned unexpected result")
}

func getProofRequireHeight(t *testing.T, f *forest, rootId primitives.BlockHeight, contract string, key string, expectedHeight int) Proof {
	root, err := f.getRootHash(rootId)
	require.NoError(t, err, "failed getting root with error: %s", err)
	proof, err := f.forest.GetProof(root, contract, key)
	f.dump(t)
	require.NoError(t, err, "failed with error: %s", err)
	require.Len(t, proof, expectedHeight, "unexpected proof length")
	return proof
}

func TestGetTopRootHash(t *testing.T) {
	f := newForest()

	rootId := updateStringEntries(f, "first", "val")
	topRoot, err1 := f.getTopRootHash()
	updatedRoot, err2 := f.getRootHash(rootId)

	require.NoError(t, err1, "GetTopHash failed with error")
	require.NoError(t, err2, "getRootHash failed with error")
	require.Equal(t, updatedRoot, topRoot, "getTopRootHash did not match getRootHash with rootId %v", rootId)
}

func TestGetPastRootHash(t *testing.T) {
	f := newForest()

	updateStringEntries(f, "first", "val")
	topRootOf1, err1 := f.getTopRootHash()
	updateStringEntries(f, "second", "val")
	rootOfOneAfterSecondUpdate, err2 := f.getRootHash(1)

	require.NoError(t, err1, "GetTopHash failed in first call")
	require.NoError(t, err2, "GetTopHash failed in second call")
	require.Equal(t, rootOfOneAfterSecondUpdate, topRootOf1, "getRootHash did not return expected hash")
}

func TestRootChangeAfterStateChange(t *testing.T) {
	f := newForest()

	updateStringEntries(f, "first", "val")
	topRootOf1, err1 := f.getTopRootHash()
	updateStringEntries(f, "first", "val1")
	topRootOf2, err2 := f.getTopRootHash()

	require.NoError(t, err1, "GetTopHash failed in first call")
	require.NoError(t, err2, "GetTopHash failed in second call")
	require.NotEqual(t, topRootOf1, topRootOf2, "root hash did not change after state change")
}

func TestRevertingStateChangeRevertsMerkleRoot(t *testing.T) {
	f := newForest()

	updateStringEntries(f, "first", "val")
	topRootOf1, err1 := f.getTopRootHash()
	updateStringEntries(f, "first", "val1")
	updateStringEntries(f, "first", "val")
	topRootOf3, err2 := f.getTopRootHash()

	require.NoError(t, err1, "GetTopHash failed in first call")
	require.NoError(t, err2, "GetTopHash failed in second call")
	require.Equal(t, topRootOf1, topRootOf3, "root hash did not revert back after resetting state")
}

func TestValidProofForMissingKey(t *testing.T) {
	f := newForest()
	key := "imNotHere"
	contract := "foo"
	proof := getProofRequireHeight(t, f, 0, contract, key, 1)
	verifyProof(t, f, 0, proof, contract, key, "", true)
	verifyProof(t, f, 0, proof, contract, key, "non-zero", false)

}

func TestAddSingleEntryToEmptyTree(t *testing.T) {
	f := newForest()
	rootId := updateStringEntries(f, "bar", "baz")
	require.EqualValues(t, 1, rootId, "unexpected root id")

	getProofRequireHeight(t, f, rootId, "", "bar", 1)
}

func TestProofValidationAfterBatchStateUpdate(t *testing.T) {
	f := newForest()

	r1 := builders.ContractStateDiff().WithContractName("foo").
		WithStringRecord("bar1", "baz").WithStringRecord("shared", "quux1").Build()
	iterator1 := r1.StateDiffsIterator()
	bar1 := iterator1.NextStateDiffs()
	shared1 := iterator1.NextStateDiffs()

	r2 := builders.ContractStateDiff().WithContractName("foo").
		WithStringRecord("bar2", "qux").WithStringRecord("shared", "quux2").Build()
	iterator2 := r2.StateDiffsIterator()
	bar2 := iterator2.NextStateDiffs()
	shared2 := iterator2.NextStateDiffs()

	root0, _ := f.getTopRootHash()
	root1 := f.Update(root0, []*protocol.ContractStateDiff{r1})
	f.Update(root1, []*protocol.ContractStateDiff{r2})

	proof := getProofRequireHeight(t, f, 1, "foo", bar1.StringKey(), 2)
	verifyProof(t, f, 1, proof, "foo", bar1.StringKey(), bar1.StringValue(), true)

	proof = getProofRequireHeight(t, f, 1, "foo", bar2.StringKey(), 2)
	verifyProof(t, f, 1, proof, "foo", bar2.StringKey(), bar2.StringValue(), false)

	proof = getProofRequireHeight(t, f, 2, "foo", bar2.StringKey(), 3)
	verifyProof(t, f, 2, proof, "foo", bar2.StringKey(), bar2.StringValue(), true)

	proof = getProofRequireHeight(t, f, 2, "foo", bar1.StringKey(), 3)
	verifyProof(t, f, 2, proof, "foo", bar1.StringKey(), bar1.StringValue(), true)

	proof = getProofRequireHeight(t, f, 1, "foo", shared1.StringKey(), 2)
	verifyProof(t, f, 1, proof, "foo", shared1.StringKey(), shared1.StringValue(), true)
	verifyProof(t, f, 1, proof, "foo", shared1.StringKey(), shared2.StringValue(), false)

	proof = getProofRequireHeight(t, f, 2, "foo", shared2.StringKey(), 2)
	verifyProof(t, f, 2, proof, "foo", shared2.StringKey(), shared2.StringValue(), true)
	verifyProof(t, f, 2, proof, "foo", shared2.StringKey(), shared1.StringValue(), false)

}

func TestProofValidationForTwoRevisionsOfSameKey(t *testing.T) {
	f := newForest()
	rootId := updateStringEntries(f, "bar1", "baz1", "bar1", "baz2")

	proof := getProofRequireHeight(t, f, rootId-1, "", "bar1", 1)
	verifyProof(t, f, rootId-1, proof, "", "bar1", "baz1", true)

	proof = getProofRequireHeight(t, f, rootId, "", "bar1", 1)
	verifyProof(t, f, rootId, proof, "", "bar1", "baz2", true)
}

func TestExtendingLeafNodeWithNoBranchesAndNoValue(t *testing.T) {
	f := newForest()
	rootId := updateStringEntries(f, "ba", "zoo", "bar", "baz", "baron", "Hello")

	getProofRequireHeight(t, f, rootId, "", "baron", 3)
}

func TestExtendingKeyPathByOneChar(t *testing.T) {
	f := newForest()
	rootId := updateStringEntries(f, "bar", "baz", "bar1", "qux")

	proof := getProofRequireHeight(t, f, rootId, "", "bar1", 2)
	verifyProof(t, f, 2, proof, "", "bar1", "qux", true)
}

func TestExtendingKeyPathBySeveralChars(t *testing.T) {
	f := newForest()

	rootId := updateStringEntries(f, "bar", "baz", "bar12", "qux", "bar123456789", "quux")

	proof := getProofRequireHeight(t, f, rootId, "", "bar123456789", 3)
	verifyProof(t, f, rootId, proof, "", "bar123456789", "quux", true)
}

func TestAddSiblingNode(t *testing.T) {
	f := newForest()
	rootId := updateStringEntries(f, "bar", "baz", "bar1", "qux", "bar2", "quux")

	proof := getProofRequireHeight(t, f, rootId, "", "bar2", 2)
	verifyProof(t, f, rootId, proof, "", "bar2", "quux", true)
}

func TestAddPathToCauseBranchingAlongExistingPath(t *testing.T) {
	f := newForest()
	rootId := updateStringEntries(f, "bar", "baz", "bar1", "qux", "bad", "quux")

	proof := getProofRequireHeight(t, f, rootId, "", "bad", 2)
	verifyProof(t, f, rootId, proof, "", "bad", "quux", true)
}

func TestReplaceExistingValueBelowDivergingPaths(t *testing.T) {
	f := newForest()
	rootId := updateStringEntries(f, "bar", "baz", "bar1", "qux", "bad", "quux", "bar1", "zoo")

	proof := getProofRequireHeight(t, f, rootId, "", "bar1", 3)
	verifyProof(t, f, rootId, proof, "", "bar1", "zoo", true)
	verifyProof(t, f, rootId, proof, "", "bar1", "qux", false)
}

func TestAddPathToCauseNewLeafAlongExistingPath(t *testing.T) {
	f := newForest()

	rootId := updateStringEntries(f, "baron", "Hirsch", "bar", "Hello")

	proof := getProofRequireHeight(t, f, rootId, "", "bar", 1)
	verifyProof(t, f, rootId, proof, "", "bar", "Hello", true)

	proof = getProofRequireHeight(t, f, rootId, "", "baron", 2)
	verifyProof(t, f, rootId, proof, "", "baron", "Hirsch", true)
}

func TestRemoveValue_SingleExistingNode(t *testing.T) {
	f := newForest()

	updateStringEntries(f, "aKey", "aValue")
	updateStringEntries(f, "aKey", "")

	baseHash, _ := f.getRootHash(0)
	topRootHash, _ := f.getTopRootHash()

	getProofRequireHeight(t, f, 0, "", "aKey", 1)
	getProofRequireHeight(t, f, 1, "", "aKey", 1)
	getProofRequireHeight(t, f, 2, "", "aKey", 1)
	require.EqualValues(t, baseHash, topRootHash, "root hash should be identical")
}

func TestRemoveValue_RemoveSingleChildLeaf(t *testing.T) {
	f := newForest()

	baseHash, _ := f.getRootHash(updateStringEntries(f, "prefix", "1"))
	updateStringEntries(f, "prefixSuffix", "2")
	topRootHash, _ := f.getRootHash(updateStringEntries(f, "prefixSuffix", ""))

	getProofRequireHeight(t, f, 1, "", "prefixSuffix", 1)
	getProofRequireHeight(t, f, 2, "", "prefixSuffix", 2)
	getProofRequireHeight(t, f, 3, "", "prefixSuffix", 1)
	require.EqualValues(t, baseHash, topRootHash, "root hash should be identical")
}

func TestRemoveValue_ParentWithSingleChild(t *testing.T) {
	f := newForest()

	updateStringEntries(f, "no", "1", "noam", "1")
	updateStringEntries(f, "no", "")

	p := getProofRequireHeight(t, f, 3, "", "noam", 1)
	require.EqualValues(t, "noam", p[0].path, "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_NonBranchingNonLeaf1(t *testing.T) {
	f := newForest()

	fullTree := updateStringEntries(f, "a", "1", "and", "2", "android", "3")
	afterRemove := updateStringEntries(f, "and", "")

	p1 := getProofRequireHeight(t, f, fullTree, "", "and", 2)
	p2 := getProofRequireHeight(t, f, afterRemove, "", "and", 2)

	getProofRequireHeight(t, f, fullTree, "", "android", 3)
	getProofRequireHeight(t, f, afterRemove, "", "android", 2)

	require.EqualValues(t, "d", p1[1].path, "full tree proof for and does not end with expected node path")
	require.EqualValues(t, "droid", p2[1].path, "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_NonBranchingNonLeaf2(t *testing.T) {
	f := newForest()

	fullTree := updateStringEntries(f, "an", "1", "and", "2", "android", "3")
	afterRemove := updateStringEntries(f, "and", "")

	p1 := getProofRequireHeight(t, f, fullTree, "", "and", 2)
	p2 := getProofRequireHeight(t, f, afterRemove, "", "and", 2)

	getProofRequireHeight(t, f, fullTree, "", "android", 3)
	getProofRequireHeight(t, f, afterRemove, "", "android", 2)

	require.EqualValues(t, "", p1[1].path, "full tree proof for and does not end with expected node path")
	require.EqualValues(t, "roid", p2[1].path, "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_BranchingNonLeaf_NodeStructureUnchanged(t *testing.T) {
	f := newForest()

	fullTree := updateStringEntries(f, "and", "1", "andalusian", "1", "android", "1")
	afterRemove := updateStringEntries(f, "and", "")

	p1 := getProofRequireHeight(t, f, afterRemove, "", "andalusian", 2)
	p2 := getProofRequireHeight(t, f, afterRemove, "", "android", 2)

	getProofRequireHeight(t, f, fullTree, "", "android", 2)
	getProofRequireHeight(t, f, afterRemove, "", "android", 2)

	require.EqualValues(t, "lusian", p1[1].path, "full tree proof for and does not end with expected node path")
	require.EqualValues(t, "oid", p2[1].path, "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_BranchingNonLeaf_CollapseBranch(t *testing.T) {
	f := newForest()

	updateStringEntries(f, "no", "7", "noam", "8", "noan", "9")
	updateStringEntries(f, "no", "")

	p0 := getProofRequireHeight(t, f, 3, "", "noam", 3)
	require.EqualValues(t, "no", p0[0].path, "unexpected proof structure")

	p := getProofRequireHeight(t, f, 4, "", "noam", 2)
	require.EqualValues(t, false, p[0].hasValue(), "unexpected proof structure")
	require.EqualValues(t, "noa", p[0].path, "unexpected proof structure")
}

func TestRemoveValue_OneOfTwoChildren(t *testing.T) {
	f := newForest()

	updateStringEntries(f, "noa", "1", "noam", "1", "noan", "1")
	updateStringEntries(f, "noan", "")

	p := getProofRequireHeight(t, f, 4, "", "noam", 2)
	getProofRequireHeight(t, f, 4, "", "noan", 1)
	require.EqualValues(t, "noa", p[0].path, "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_OneOfTwoChildrenCollapsingParent(t *testing.T) {
	f := newForest()

	updateStringEntries(f, "noam", "8", "noan", "9")
	updateStringEntries(f, "noan", "")

	p := getProofRequireHeight(t, f, 3, "", "noam", 1)
	getProofRequireHeight(t, f, 3, "", "noan", 1)
	require.EqualValues(t, "noam", p[0].path, "unexpected proof structure")
}

func TestRemoveValue_MissingKey(t *testing.T) {
	f := newForest()

	baseHash, err := f.getRootHash(updateStringEntries(f, "noam", "1", "noan", "1", "noamon", "1", "noamiko", "1"))
	hash1, err1 := f.getRootHash(updateStringEntries(f, "noamiko_andSomeSuffix", ""))
	hash2, err2 := f.getRootHash(updateStringEntries(f, "noa", ""))
	hash3, err3 := f.getRootHash(updateStringEntries(f, "n", ""))
	hash4, err4 := f.getRootHash(updateStringEntries(f, "noamo", ""))

	require.NoError(t, err, "unexpected error")
	require.NoError(t, err1, "unexpected error")
	require.NoError(t, err2, "unexpected error")
	require.NoError(t, err3, "unexpected error")
	require.NoError(t, err4, "unexpected error")
	require.EqualValues(t, baseHash, hash1, "tree changed after removing missing key")
	require.EqualValues(t, baseHash, hash2, "tree changed after removing missing key")
	require.EqualValues(t, baseHash, hash3, "tree changed after removing missing key")
	require.EqualValues(t, baseHash, hash4, "tree changed after removing missing key")
}

func TestOrderOfAdditionsDoesNotMatter(t *testing.T) {
	keyValue := []string{"bar", "baz", "bar123", "qux", "bar1234", "quux", "bad", "foo", "bank", "hello"}
	var1 := []int{2, 6, 0, 8, 4}
	var2 := []int{8, 4, 0, 2, 6}
	var3 := []int{8, 6, 4, 2, 0}

	f1 := newForest()
	rootId1 := updateStringEntries(f1, keyValue[var1[0]], keyValue[var1[0]+1], keyValue[var1[1]], keyValue[var1[1]+1],
		keyValue[var1[2]], keyValue[var1[2]+1], keyValue[var1[3]], keyValue[var1[3]+1], keyValue[var1[4]], keyValue[var1[4]+1])
	root1, _ := f1.getRootHash(rootId1)
	proof1, _ := f1.forest.GetProof(root1, "", "bar1234")

	f2 := newForest()
	rootId2 := updateStringEntries(f2, keyValue[var2[0]], keyValue[var2[0]+1], keyValue[var2[1]], keyValue[var2[1]+1],
		keyValue[var2[2]], keyValue[var2[2]+1], keyValue[var2[3]], keyValue[var2[3]+1], keyValue[var2[4]], keyValue[var2[4]+1])
	root2, _ := f2.getRootHash(rootId2)
	proof2, _ := f2.forest.GetProof(root2, "", "bar1234")

	require.Equal(t, rootId1, rootId2, "unexpected different rootId")
	require.Equal(t, root1, root2, "unexpected different root hash")
	require.Equal(t, len(proof1), len(proof2), "unexpected different tree depth / proof lengths")
	require.Equal(t, proof1[3].hash(), proof2[3].hash(), "unexpected different leaf node hash")

	f3 := newForest()
	rootId3 := updateStringEntries(f3, keyValue[var3[0]], keyValue[var3[0]+1], keyValue[var3[1]], keyValue[var3[1]+1],
		keyValue[var3[2]], keyValue[var3[2]+1], keyValue[var3[3]], keyValue[var3[3]+1], keyValue[var3[4]], keyValue[var3[4]+1])
	root3, _ := f3.getRootHash(rootId3)
	proof3, _ := f3.forest.GetProof(root3, "", "bar1234")

	require.Equal(t, rootId2, rootId3, "unexpected different rootId")
	require.Equal(t, root2, root3, "unexpected different root hash")
	require.Equal(t, len(proof2), len(proof3), "unexpected different tree depth / proof lengths")
	require.Equal(t, proof2[3].hash(), proof3[3].hash(), "unexpected different leaf node hash")
}

// =================
// Debug helpers
// =================

func (f *forest) dump(t *testing.T) {
	t.Logf("---------------- TRIE BEGIN ------------------")
	for i, h := range f.roots {
		label := " Ω"
		if int(i) == len(f.roots)-1 {
			label = "*Ω"
		}
		f.forest.nodes[h.KeyForMap()].printNode(label, 0, f.forest, t)
	}
	t.Logf("---------------- TRIE END --------------------")
}

func (n *Node) printNode(label string, depth int, trie *Forest, t *testing.T) {
	prefix := strings.Repeat(" ", depth)
	leafText := ""
	if n.hasValue() {
		leafText = fmt.Sprintf(": %v", n.value)
	}
	pathString := fmt.Sprintf("%s%s)%s", prefix, label, n.path)
	t.Logf("%s%s\n", pathString, leafText)
	for l, v := range n.branches {
		if len(v) != 0 {
			trie.nodes[v.KeyForMap()].printNode(string([]byte{byte(l)}), depth+len(pathString)-1, trie, t)
		}
	}
}

func (p *Proof) dump() {
	fmt.Println("---------------- PROOF BEGIN ------------------")
	for _, n := range *p {
		hash2 := n.hash()
		fmt.Printf("%s\n%+v\n", base64.StdEncoding.EncodeToString(hash2[:]), n)
	}
	fmt.Println("---------------- PROOF END --------------------")
}

// TODO - this just checks there are no data integrity in our forest integrity
func (f *forest) testForestIntegrity(t *testing.T) {
	for h, n := range f.forest.nodes {
		require.Equal(t, n.hash().KeyForMap(), h, "node key is not true hash code")
	}
	for _, root := range f.roots {
		require.Contains(t, f.forest.nodes, root.KeyForMap(), "missing child node")
	}
	require.Equal(t, f.roots[primitives.BlockHeight(len(f.roots))-1], f.top, "top root is not the most recent root")
}
