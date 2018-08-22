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
)

func updateStringEntries(f *Forest, keyValues ...string) TrieId {
	if len(keyValues)%2 != 0 {
		panic("expected key value pairs")
	}
	for i := 0; i < len(keyValues); i = i + 2 {
		f.updateSingleEntry(keyValues[i], hash.CalcSha256([]byte(keyValues[i+1])))
	}
	return f.topRoot
}

func verifyProof(t *testing.T, f *Forest, trieId TrieId, proof Proof, contract string, key string, value string, exists bool) {
	rootHash, _ := f.GetRootHash(trieId)
	verified, err := f.Verify(rootHash, proof, contract, key, value)
	require.NoError(t, err, "proof verification failed")
	require.Equal(t, exists, verified, "proof verification returned unexpected result")
}

func getProofRequireHeight(t *testing.T, f *Forest, rootId TrieId, contract string, key string, expectedHeight int) Proof {
	proof, err := f.GetProof(rootId, contract, key)
	require.NoError(t, err, "failed with error: %s", err)
	require.Len(t, proof, expectedHeight, "unexpected proof length of %v found %v", expectedHeight, len(proof))
	return proof
}

func TestGetTopRootHash(t *testing.T) {
	f := NewForest()

	rootId := updateStringEntries(f, "first", "val")
	topRoot, err1 := f.GetTopRootHash()
	updatedRoot, err2 := f.GetRootHash(rootId)

	require.NoError(t, err1, "GetTopHash failed with error")
	require.NoError(t, err2, "GetRootHash failed with error")
	require.Equal(t, updatedRoot, topRoot, "GetTopRootHash did not match GetRootHash with rootId %v", rootId)
}

func TestGetPastRootHash(t *testing.T) {
	f := NewForest()

	updateStringEntries(f, "first", "val")
	topRootOf1, err1 := f.GetTopRootHash()
	updateStringEntries(f, "second", "val")
	rootOfOneAfterSecondUpdate, err2 := f.GetRootHash(1)

	require.NoError(t, err1, "GetTopHash failed in first call")
	require.NoError(t, err2, "GetTopHash failed in second call")
	require.Equal(t, rootOfOneAfterSecondUpdate, topRootOf1, "GetRootHash did not return expected hash")
}

func TestRootChangeAfterStateChange(t *testing.T) {
	f := NewForest()

	updateStringEntries(f, "first", "val")
	topRootOf1, err1 := f.GetTopRootHash()
	updateStringEntries(f, "first", "val1")
	topRootOf2, err2 := f.GetTopRootHash()

	require.NoError(t, err1, "GetTopHash failed in first call")
	require.NoError(t, err2, "GetTopHash failed in second call")
	require.NotEqual(t, topRootOf1, topRootOf2, "root hash did not change after state change")
}

func TestRevertingStateChangeRevertsMerkleRoot(t *testing.T) {
	f := NewForest()

	updateStringEntries(f, "first", "val")
	topRootOf1, err1 := f.GetTopRootHash()
	updateStringEntries(f, "first", "val1")
	updateStringEntries(f, "first", "val")
	topRootOf3, err2 := f.GetTopRootHash()

	require.NoError(t, err1, "GetTopHash failed in first call")
	require.NoError(t, err2, "GetTopHash failed in second call")
	require.Equal(t, topRootOf1, topRootOf3, "root hash did not revert back after resetting state")
}

func TestValidProofForMissingKey(t *testing.T) {
	f := NewForest()
	key := "imNotHere"
	contract := "foo"
	proof := getProofRequireHeight(t, f, 0, contract, key, 1)
	verifyProof(t, f, 0, proof, contract, key, "", true)
	verifyProof(t, f, 0, proof, contract, key, "non-zero", false)

}

func TestAddSingleEntryToEmptyTree(t *testing.T) {
	f := NewForest()
	rootId := updateStringEntries(f, "bar", "baz")
	require.Equal(t, TrieId(1), rootId, "unexpected root id")

	getProofRequireHeight(t, f, rootId, "", "bar", 1)
}

func TestProofValidationAfterBatchStateUpdate(t *testing.T) {
	f := NewForest()
	diffContract := builders.ContractStateDiff().WithContractName("foo")
	r1 := diffContract.WithStringRecord("bar1", "baz").Build()
	k1 := r1.StateDiffsIterator().NextStateDiffs().StringKey()
	v1 := r1.StateDiffsIterator().NextStateDiffs().StringValue()
	f.Update([]*protocol.ContractStateDiff{r1})

	diffContract = builders.ContractStateDiff().WithContractName("foo")
	r2 := diffContract.WithStringRecord("bar2", "qux").Build()
	k2 := r2.StateDiffsIterator().NextStateDiffs().StringKey()
	v2 := r2.StateDiffsIterator().NextStateDiffs().StringValue()
	f.Update([]*protocol.ContractStateDiff{r2})

	proof := getProofRequireHeight(t, f, 1, "foo", k1, 1)
	verifyProof(t, f, 1, proof, "foo", k1, v1, true)

	proof = getProofRequireHeight(t, f, 1, "foo", k2, 1)
	verifyProof(t, f, 1, proof, "foo", k2, v2, false)

	proof = getProofRequireHeight(t, f, 2, "foo", k2, 2)
	verifyProof(t, f, 2, proof, "foo", k2, v2, true)

	proof = getProofRequireHeight(t, f, 2, "foo", k1, 2)
	verifyProof(t, f, 2, proof, "foo", k1, v1, true)
}

func TestProofValidationForTwoRevisionsOfSameKey(t *testing.T) {
	f := NewForest()
	rootId := updateStringEntries(f, "bar1", "baz1", "bar1", "baz2")

	proof := getProofRequireHeight(t, f, rootId-1, "", "bar1", 1)
	verifyProof(t, f, rootId-1, proof, "", "bar1", "baz1", true)

	proof = getProofRequireHeight(t, f, rootId, "", "bar1", 1)
	verifyProof(t, f, rootId, proof, "", "bar1", "baz2", true)
}

func TestExtendingLeafNodeWithNoBranchesAndNoValue(t *testing.T) {
	f := NewForest()
	rootId := updateStringEntries(f, "ba", "zoo", "bar", "baz", "baron", "Hello")

	getProofRequireHeight(t, f, rootId, "", "baron", 3)
}

func TestExtendingKeyPathByOneChar(t *testing.T) {
	f := NewForest()
	rootId := updateStringEntries(f, "bar", "baz", "bar1", "qux")

	proof := getProofRequireHeight(t, f, rootId, "", "bar1", 2)
	verifyProof(t, f, 2, proof, "", "bar1", "qux", true)
}

func TestExtendingKeyPathBySeveralChars(t *testing.T) {
	f := NewForest()

	rootId := updateStringEntries(f, "bar", "baz", "bar12", "qux", "bar123456789", "quux")

	proof := getProofRequireHeight(t, f, rootId, "", "bar123456789", 3)
	verifyProof(t, f, rootId, proof, "", "bar123456789", "quux", true)
}

func TestAddSiblingNode(t *testing.T) {
	f := NewForest()
	rootId := updateStringEntries(f, "bar", "baz", "bar1", "qux", "bar2", "quux")

	proof := getProofRequireHeight(t, f, rootId, "", "bar2", 2)
	verifyProof(t, f, rootId, proof, "", "bar2", "quux", true)
}

func TestAddPathToCauseBranchingAlongExistingPath(t *testing.T) {
	f := NewForest()
	rootId := updateStringEntries(f, "bar", "baz", "bar1", "qux", "bad", "quux")

	proof := getProofRequireHeight(t, f, rootId, "", "bad", 2)
	verifyProof(t, f, rootId, proof, "", "bad", "quux", true)
}

func TestReplaceExistingValueBelowDivergingPaths(t *testing.T) {
	f := NewForest()
	rootId := updateStringEntries(f, "bar", "baz", "bar1", "qux", "bad", "quux", "bar1", "zoo")

	proof := getProofRequireHeight(t, f, rootId, "", "bar1", 3)
	verifyProof(t, f, rootId, proof, "", "bar1", "zoo", true)
	verifyProof(t, f, rootId, proof, "", "bar1", "qux", false)
}

func TestAddPathToCauseNewLeafAlongExistingPath(t *testing.T) {
	f := NewForest()

	rootId := updateStringEntries(f, "baron", "Hirsch", "bar", "Hello")

	proof := getProofRequireHeight(t, f, rootId, "", "bar", 1)
	verifyProof(t, f, rootId, proof, "", "bar", "Hello", true)

	proof = getProofRequireHeight(t, f, rootId, "", "baron", 2)
	verifyProof(t, f, rootId, proof, "", "baron", "Hirsch", true)
}

func TestRemoveValue_SingleExistingNode(t *testing.T) {
	f := NewForest()

	updateStringEntries(f, "aKey", "aValue")
	updateStringEntries(f, "aKey", "")

	baseHash, _ := f.GetRootHash(0)
	topRootHash, _ := f.GetTopRootHash()

	getProofRequireHeight(t, f, 0, "", "aKey", 1)
	getProofRequireHeight(t, f, 1, "", "aKey", 1)
	getProofRequireHeight(t, f, 2, "", "aKey", 1)
	require.EqualValues(t, baseHash, topRootHash, "root hash should be identical")
}

func TestRemoveValue_RemoveSingleChildLeaf(t *testing.T) {
	f := NewForest()

	baseHash, _ := f.GetRootHash(updateStringEntries(f, "prefix", "1"))
	updateStringEntries(f, "prefixSuffix", "2")
	topRootHash, _ := f.GetRootHash(updateStringEntries(f, "prefixSuffix", ""))

	getProofRequireHeight(t, f, 1, "", "prefixSuffix", 1)
	getProofRequireHeight(t, f, 2, "", "prefixSuffix", 2)
	getProofRequireHeight(t, f, 3, "", "prefixSuffix", 1)
	require.EqualValues(t, baseHash, topRootHash, "root hash should be identical")
}

func TestRemoveValue_ParentWithSingleChild(t *testing.T) {
	f := NewForest()

	updateStringEntries(f, "no", "1", "noam", "1")
	updateStringEntries(f, "no", "")

	p := getProofRequireHeight(t, f, 3, "", "noam", 1)
	require.EqualValues(t, "noam", p[0].path, "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_NonBranchingNonLeaf1(t *testing.T) {
	f := NewForest()

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
	f := NewForest()

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
	f := NewForest()

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
	f := NewForest()

	updateStringEntries(f, "no", "7", "noam", "8", "noan", "9")
	updateStringEntries(f, "no", "")

	p0 := getProofRequireHeight(t, f, 3, "", "noam", 3)
	require.EqualValues(t, "no", p0[0].path, "unexpected proof structure")

	p := getProofRequireHeight(t, f, 4, "", "noam", 2)
	require.EqualValues(t, false, p[0].hasValue(), "unexpected proof structure")
	require.EqualValues(t, "noa", p[0].path, "unexpected proof structure")
}

func TestRemoveValue_OneOfTwoChildren(t *testing.T) {
	f := NewForest()

	updateStringEntries(f, "noa", "1", "noam", "1", "noan", "1")
	updateStringEntries(f, "noan", "")

	p := getProofRequireHeight(t, f, 4, "", "noam", 2)
	getProofRequireHeight(t, f, 4, "", "noan", 1)
	require.EqualValues(t, "noa", p[0].path, "full tree proof for and does not end with expected node path")
}

func TestRemoveValue_OneOfTwoChildrenCollapsingParent(t *testing.T) {
	f := NewForest()

	updateStringEntries(f, "noam", "8", "noan", "9")
	updateStringEntries(f, "noan", "")

	p := getProofRequireHeight(t, f, 3, "", "noam", 1)
	getProofRequireHeight(t, f, 3, "", "noan", 1)
	require.EqualValues(t, "noam", p[0].path, "unexpected proof structure")
}

func TestRemoveValue_MissingKey(t *testing.T) {
	f := NewForest()

	baseHash, err := f.GetRootHash(updateStringEntries(f, "noam", "1", "noan", "1", "noamon", "1", "noamiko", "1"))
	hash1, err1 := f.GetRootHash(updateStringEntries(f, "noamiko_andSomeSuffix", ""))
	hash2, err2 := f.GetRootHash(updateStringEntries(f, "noa", ""))
	hash3, err3 := f.GetRootHash(updateStringEntries(f, "n", ""))
	hash4, err4 := f.GetRootHash(updateStringEntries(f, "noamo", ""))

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

//TODO - updateStringEntries should advance TrieId only by one
//TODO - updateStringEntries - the bulk update version (optimize node access)
//TODO - Radix 16
//TODO - parity
//TODO - use hashes of contract names
//TODO - GetProof - accept an in memory list of cached nodes (to support bulk proof fetch).
//TODO - serialization based on spec
//TODO - split branch and node leafs (this can be limited to serializeation only)
//TODO - accept Node DB object
//TODO - garbage collection
//TODO - avoid hashing values of less than 32 bytes
//TODO - what hash functions should be used for values and what functions for node addresses?
//TODO - in case save key length is enforced - accept a key length in the forest constructor
//TODO - Prepare for GC (set values of older nodes to know when they were last valid)

//TODO - change verify and update types to []byte from strings

// Debug helpers
// TODO - we don't use any of these. but they are useful for debugging

func TestOrderOfAdditionsDoesNotMatter(t *testing.T) {
	keyValue := []string{"bar", "baz", "bar123", "qux", "bar1234", "quux", "bad", "foo", "bank", "hello"}
	var1 := []int{2, 6, 0, 8, 4}
	var2 := []int{8, 4, 0, 2, 6}
	var3 := []int{8, 6, 4, 2, 0}

	f1 := NewForest()
	rootId1 := updateStringEntries(f1, keyValue[var1[0]], keyValue[var1[0]+1], keyValue[var1[1]], keyValue[var1[1]+1],
		keyValue[var1[2]], keyValue[var1[2]+1], keyValue[var1[3]], keyValue[var1[3]+1], keyValue[var1[4]], keyValue[var1[4]+1])
	root1, _ := f1.GetRootHash(rootId1)
	proof1, _ := f1.GetProof(rootId1, "", "bar1234")

	f2 := NewForest()
	rootId2 := updateStringEntries(f2, keyValue[var2[0]], keyValue[var2[0]+1], keyValue[var2[1]], keyValue[var2[1]+1],
		keyValue[var2[2]], keyValue[var2[2]+1], keyValue[var2[3]], keyValue[var2[3]+1], keyValue[var2[4]], keyValue[var2[4]+1])
	root2, _ := f2.GetRootHash(rootId2)
	proof2, _ := f2.GetProof(rootId2, "", "bar1234")

	require.Equal(t, rootId1, rootId2, "unexpected different rootId")
	require.Equal(t, root1, root2, "unexpected different root hash")
	require.Equal(t, len(proof1), len(proof2), "unexpected different tree depth / proof lengths")
	require.Equal(t, proof1[3].hash(), proof2[3].hash(), "unexpected different leaf node hash")

	f3 := NewForest()
	rootId3 := updateStringEntries(f3, keyValue[var3[0]], keyValue[var3[0]+1], keyValue[var3[1]], keyValue[var3[1]+1],
		keyValue[var3[2]], keyValue[var3[2]+1], keyValue[var3[3]], keyValue[var3[3]+1], keyValue[var3[4]], keyValue[var3[4]+1])
	root3, _ := f3.GetRootHash(rootId3)
	proof3, _ := f3.GetProof(rootId3, "", "bar1234")

	require.Equal(t, rootId2, rootId3, "unexpected different rootId")
	require.Equal(t, root2, root3, "unexpected different root hash")
	require.Equal(t, len(proof2), len(proof3), "unexpected different tree depth / proof lengths")
	require.Equal(t, proof2[3].hash(), proof3[3].hash(), "unexpected different leaf node hash")
}

func (f *Forest) dump() {
	fmt.Println("---------------- TRIE BEGIN ------------------")
	for i, h := range f.roots {
		label := " Ω"
		if int(i) == len(f.roots)-1 {
			label = "*Ω"
		}
		f.nodes[h.KeyForMap()].printNode(label, 0, f)
	}
	fmt.Println("---------------- TRIE END --------------------")
}

func (n *Node) printNode(label string, depth int, trie *Forest) {
	prefix := strings.Repeat(" ", depth)
	leafText := ""
	if n.hasValue() {
		leafText = fmt.Sprintf(": %v", n.value)
	}
	pathString := fmt.Sprintf("%s%s)%s", prefix, label, n.path)
	fmt.Printf("%s%s\n", pathString, leafText)
	for l, v := range n.branches {
		if len(v) != 0 {
			trie.nodes[v.KeyForMap()].printNode(string([]byte{byte(l)}), depth+len(pathString)-1, trie)
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
func (f *Forest) testForestIntegrity(t *testing.T) {
	for h, n := range f.nodes {
		require.Equal(t, n.hash().KeyForMap(), h, "node key is not true hash code")
	}
	for _, root := range f.roots {
		require.Contains(t, f.nodes, root.KeyForMap(), "missing child node")
	}
	require.Equal(t, f.roots[TrieId(len(f.roots))-1], f.topRoot, "top root is not the most recent root")
}
