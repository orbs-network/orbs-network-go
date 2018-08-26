package test

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPersistStateToStorage(t *testing.T) {
	d := newStateStorageDriver(1)

	contract1 := builders.ContractStateDiff().WithContractName("contract1").WithStringRecord("key1", "v1").WithStringRecord("key2", "v2").Build()
	contract2 := builders.ContractStateDiff().WithContractName("contract2").WithStringRecord("key1", "v3").Build()

	d.commitStateDiff(CommitStateDiff().WithBlockHeight(1).WithDiff(contract1).WithDiff(contract2).Build())

	output, err := d.readSingleKey("contract1", "key1")
	require.NoError(t, err)
	require.EqualValues(t, "v1", output, "unexpected value read from storage")
	output2, err := d.readSingleKey("contract1", "key2")
	require.NoError(t, err)
	require.EqualValues(t, "v2", output2, "unexpected value read from storage")
	output3, err := d.readSingleKey("contract2", "key1")
	require.NoError(t, err)
	require.EqualValues(t, "v3", output3, "unexpected value read from storage")
}

func TestNonConsecutiveBlockHeights(t *testing.T) {
	d := newStateStorageDriver(1)

	registerContractDiff := builders.ContractStateDiff().WithContractName("contract1").WithStringRecord("key1", "whatever").Build()
	d.service.CommitStateDiff(CommitStateDiff().WithBlockHeight(1).WithDiff(registerContractDiff).Build())

	diff := builders.ContractStateDiff().WithContractName("contract1").WithStringRecord("key1", "whatever").Build()
	result, err := d.service.CommitStateDiff(CommitStateDiff().WithBlockHeight(3).WithDiff(diff).Build())

	require.NoError(t, err)
	require.EqualValues(t, 2, result.NextDesiredBlockHeight, "unexpected NextDesiredBlockHeight")

	_, err = d.readSingleKey("contract1", "key1")
	require.NoError(t, err)
}

func TestCommitPastBlockHeights(t *testing.T) {
	d := newStateStorageDriver(1)
	v1 := "v1"
	v2 := "v2"

	contractDiff := builders.ContractStateDiff().WithContractName("contract1")
	diffAtHeight1 := CommitStateDiff().WithBlockHeight(1).WithDiff(contractDiff.WithStringRecord("key1", v1).Build()).Build()
	diffAtHeight2 := CommitStateDiff().WithBlockHeight(2).WithDiff(contractDiff.WithStringRecord("key1", v2).Build()).Build()

	d.service.CommitStateDiff(diffAtHeight1)
	d.service.CommitStateDiff(diffAtHeight2)

	diffWrongOldHeight := CommitStateDiff().WithBlockHeight(1).WithDiff(contractDiff.WithStringRecord("key1", "v3").WithStringRecord("key3", "v3").Build()).Build()
	result, err := d.service.CommitStateDiff(diffWrongOldHeight)
	require.NoError(t, err)
	require.EqualValues(t, 3, result.NextDesiredBlockHeight, "unexpected NextDesiredBlockHeight")

	output, err := d.readSingleKeyFromRevision(2, "contract1", "key1")
	require.NoError(t, err)
	require.EqualValues(t, v2, output, "unexpected value read")
	output2, err := d.readSingleKeyFromRevision(2, "contract1", "key3")
	require.NoError(t, err)
	require.EqualValues(t, []byte{}, output2, "unexpected value read")
}
