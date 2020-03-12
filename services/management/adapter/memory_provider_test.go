package adapter

import (
	"testing"
)

func TestManagementMemory_PreventDoubleCommitteeOnSameBlock(t *testing.T) {
	//with.Logging(t, func(harness *with.LoggingHarness) {
	//	cp := newProvider(testKeys.NodeAddressesForTests()[:4], harness.Logger)
	//	termChangeHeight := uint64(10)
	//	err := cp.SetCommitteeToTestKeysWithIndices(termChangeHeight, 1, 2, 3, 4)
	//	require.NoError(t, err)
	//
	//	err = cp.SetCommitteeToTestKeysWithIndices(termChangeHeight-1, 1, 2, 3, 4)
	//	require.Error(t, err, "must fail on smaller")
	//
	//	err = cp.SetCommitteeToTestKeysWithIndices(termChangeHeight, 1, 2, 3, 4)
	//	require.Error(t, err, "must fail on equal")
	//})
}

