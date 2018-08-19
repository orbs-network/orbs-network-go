package test

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInitToZero(t *testing.T) {
	d := newStateStorageDriver(1)
	height, timestamp, err := d.getBlockHeightAndTimestamp()

	require.NoError(t, err, "unexpected error")
	require.EqualValues(t, 0, height, "unexpected height")
	require.EqualValues(t, 0, timestamp, "unexpected timestamp")
}

func TestReflectsSuccessfulCommit(t *testing.T) {
	d := newStateStorageDriver(1)
	heightBefore, _, err := d.getBlockHeightAndTimestamp()
	d.service.CommitStateDiff(CommitStateDiff().WithBlockHeight(1).WithBlockTimestamp(6579).WithDiff(builders.ContractStateDiff().Build()).Build())
	heightAfter, timestampAfter, err := d.getBlockHeightAndTimestamp()

	require.NoError(t, err, "unexpected error")
	require.EqualValues(t, heightBefore+1, heightAfter, "unexpected height")
	require.EqualValues(t, 6579, timestampAfter, "unexpected timestamp")
}

func TestIgnoreFailedCommit(t *testing.T) {
	d := newStateStorageDriver(1)
	stateDiff := builders.ContractStateDiff().Build()
	d.service.CommitStateDiff(CommitStateDiff().WithBlockHeight(1).WithDiff(stateDiff).Build())
	d.service.CommitStateDiff(CommitStateDiff().WithBlockHeight(2).WithDiff(stateDiff).Build())
	heightBefore, _, err := d.getBlockHeightAndTimestamp()
	d.service.CommitStateDiff(CommitStateDiff().WithBlockHeight(1).WithDiff(stateDiff).Build())
	heightAfter, _, err := d.getBlockHeightAndTimestamp()

	require.NoError(t, err, "unexpected error")
	require.EqualValues(t, heightBefore, heightAfter, "unexpected height")
}
