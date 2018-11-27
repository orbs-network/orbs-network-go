package merkle

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/services/statestorage/merkle.old"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func newVerTraslateKeys(keyValues ...string) map[string]primitives.Sha256 {
	if len(keyValues)%2 != 0 {
		panic("expected key value pairs")
	}

	if len(keyValues)%2 != 0 {
		panic("expected key value pairs")
	}
	diffs := make(map[string]primitives.Sha256)
	for i := 0; i < len(keyValues); i = i + 2 {
		k := fmt.Sprintf("%s", hash.CalcRipmd160Sha256([]byte(keyValues[i])))
		s := fmt.Sprintf("%x", []byte(keyValues[i+1]))

		diffs[k] = hash.CalcSha256([]byte(s))
		//fmt.Printf("new path %s value %x\n", k, hash.CalcSha256([]byte(s)))
	}
	return diffs
}

func oldVerTraslateKeys(keyValues ...string) []*protocol.ContractStateDiff {
	if len(keyValues)%2 != 0 {
		panic("expected key value pairs")
	}

	stateDiffs := []*protocol.StateRecordBuilder{}
	for i := 0; i < len(keyValues); i = i + 2 {
		stateDiffs = append(stateDiffs, &protocol.StateRecordBuilder{
			Key:   hash.CalcRipmd160Sha256([]byte(keyValues[i])),
			Value: []byte(keyValues[i+1]),
		})
	}

	contractStateDiff := &protocol.ContractStateDiffBuilder{
		ContractName: "",
		StateDiffs:   stateDiffs,
	}
	contractStateDiffs := []*protocol.ContractStateDiff{contractStateDiff.Build()}
	return contractStateDiffs
}

var nTimes = 1000

func TestTwoVersionsOfMerkle(t *testing.T) {
	keyValues := []string{"abdbda", "1", "abdcda", "1", "acdbda", "1", "acdcda", "1", "aeeeee", "5", "bbbaaa", "4", "bcbaaa", "4", "bbbada", "4", "bbbadc", "6", "bbbade", "7"}
	diffs := newVerTraslateKeys(keyValues...)
	oldDiffs := oldVerTraslateKeys(keyValues...)

	forest, root := NewForest()
	root1 := primitives.MerkleSha256{}
	root1, _ = forest.Update(root, diffs)

	oldForest, oldRoot := merkle.NewForest()
	oldRoot1 := primitives.MerkleSha256{}
	oldRoot1, _ = oldForest.Update(oldRoot, oldDiffs)

	require.Equal(t, root, oldRoot, "unexpected different empty root hash")
	require.Equal(t, root1, oldRoot1, "unexpected different root hash")

	// stress
	start := time.Now()
	for i := 0; i < nTimes; i++ {
		forest, root := NewForest()
		root1, _ = forest.Update(root, diffs)
	}
	fmt.Printf("New implementation Took: %s\n", time.Since(start))

	start = time.Now()
	for i := 0; i < nTimes; i++ {
		oldForest, oldRoot := merkle.NewForest()
		oldRoot1, _ = oldForest.Update(oldRoot, oldDiffs)
	}
	fmt.Printf("Old implementation Took: %s\n", time.Since(start))
}
