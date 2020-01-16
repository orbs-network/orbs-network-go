package memory

import (
	"context"
	"github.com/orbs-network/lean-helix-go/test"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMemoryCommittee_GetCommitteeWhenOnlyOneTerm(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			cp := newProvider(testKeys.NodeAddressesForTests()[:4], harness.Logger)

			committee, err := cp.GetCommittee(ctx, 0)
			require.NoError(t, err)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")

			committee, err = cp.GetCommittee(ctx, 10)
			require.NoError(t, err)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")
		})
	})
}

func TestMemoryCommittee_GetCommitteeAfterAnUpdateExists(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {

			cp := newProvider(testKeys.NodeAddressesForTests()[:4], harness.Logger)
			termChangeHeight := uint64(10)
			cp.SetCommitteeToTestKeysWithIndices(termChangeHeight, 1, 2, 3, 4)

			committee, err := cp.GetCommittee(ctx, termChangeHeight-1)
			require.NoError(t, err)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")

			committee, err = cp.GetCommittee(ctx, termChangeHeight)
			require.NoError(t, err)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee, err = cp.GetCommittee(ctx, termChangeHeight+1)
			require.NoError(t, err)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")
		})
	})
}

func TestMemoryCommittee_GetCommitteeWhenTwoChangesOneAfterOther(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {

			cp := newProvider(testKeys.NodeAddressesForTests()[:4], harness.Logger)
			termChangeHeight := uint64(10)
			cp.SetCommitteeToTestKeysWithIndices(termChangeHeight, 1, 2, 3, 4)
			cp.SetCommitteeToTestKeysWithIndices(termChangeHeight+1, 5, 6, 7, 8)

			committee, err := cp.GetCommittee(ctx, termChangeHeight)
			require.NoError(t, err)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee, err = cp.GetCommittee(ctx, termChangeHeight+1)
			require.NoError(t, err)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")

			committee, err = cp.GetCommittee(ctx, termChangeHeight+2)
			require.NoError(t, err)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")
		})
	})
}

func TestMemoryCommittee_GetCommitteeWhenTwoChangesClose(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {

			cp := newProvider(testKeys.NodeAddressesForTests()[:4], harness.Logger)
			termChangeHeight := uint64(10)
			cp.SetCommitteeToTestKeysWithIndices(termChangeHeight, 1, 2, 3, 4)
			cp.SetCommitteeToTestKeysWithIndices(termChangeHeight+2, 5, 6, 7, 8)

			committee, err := cp.GetCommittee(ctx, termChangeHeight)
			require.NoError(t, err)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee, err = cp.GetCommittee(ctx, termChangeHeight+1)
			require.NoError(t, err)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee, err = cp.GetCommittee(ctx, termChangeHeight+2)
			require.NoError(t, err)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")

			committee, err = cp.GetCommittee(ctx, termChangeHeight+3)
			require.NoError(t, err)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")
		})
	})
}

func TestMemoryCommittee_PreventDoubleCommitteeOnSameBlock(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		cp := newProvider(testKeys.NodeAddressesForTests()[:4], harness.Logger)
		termChangeHeight := uint64(10)
		err := cp.SetCommitteeToTestKeysWithIndices(termChangeHeight, 1, 2, 3, 4)
		require.NoError(t, err)

		err = cp.SetCommitteeToTestKeysWithIndices(termChangeHeight-1, 1, 2, 3, 4)
		require.Error(t, err, "must fail on smaller")

		err = cp.SetCommitteeToTestKeysWithIndices(termChangeHeight, 1, 2, 3, 4)
		require.Error(t, err, "must fail on equal")
	})
}

func newProvider(committee []primitives.NodeAddress, logger log.Logger) *CommitteeProvider {
	return  &CommitteeProvider{committees: []*committeeTerm{{0, committee}}, logger:logger}
}
