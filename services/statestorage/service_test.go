package statestorage

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_inflateChainState(t *testing.T) {
	singleDiff := (&protocol.ContractStateDiffBuilder{
		ContractName: "Albums",
		StateDiffs: []*protocol.StateRecordBuilder{
			{
				Key:   []byte("David Bowie"),
				Value: []byte("Station to Station"),
			},
		},
	}).Build()

	diffs := []*protocol.ContractStateDiff{
		singleDiff,
	}

	chainState := inflateChainState(diffs)
	singleDiff.StateDiffsIterator().NextStateDiffs().MutateValue([]byte("Station of Station"))
	singleDiff.MutateContractName("Album1")

	require.NotNil(t, chainState["Albums"])
	require.EqualValues(t, []byte("Station to Station"), chainState["Albums"]["David Bowie"].Value(), "the underlying buffer was not copied")
}
