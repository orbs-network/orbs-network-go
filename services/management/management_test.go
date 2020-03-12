package management

import (
	"context"
	"github.com/orbs-network/lean-helix-go/test"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestManagement_GetCommitteeWhenOnlyOneTerm(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			cp := newStaticCommitteeManagement(0, testKeys.NodeAddressesForTests()[:4])

			committee := cp.GetCommittee(ctx, 0)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")

			committee = cp.GetCommittee(ctx, 10)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")
		})
	})
}

func TestManagement_GetCommitteeAfterAnUpdateExists(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			cp := newStaticCommitteeManagement(0, testKeys.NodeAddressesForTests()[:4])
			termChangeHeight := uint64(10)
			cp.addCommittee(termChangeHeight, testKeys.NodeAddressesForTests()[1:5])

			committee := cp.GetCommittee(ctx, termChangeHeight-1)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")

			committee = cp.GetCommittee(ctx, termChangeHeight)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee = cp.GetCommittee(ctx, termChangeHeight+1)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")
		})
	})
}

func TestManagement_GetCommitteeWhenTwoChangesOneAfterOther(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			cp := newStaticCommitteeManagement(0, testKeys.NodeAddressesForTests()[:4])
			termChangeHeight := uint64(10)
			cp.addCommittee(termChangeHeight, testKeys.NodeAddressesForTests()[1:5])
			cp.addCommittee(termChangeHeight+1, testKeys.NodeAddressesForTests()[5:9])

			committee := cp.GetCommittee(ctx, termChangeHeight)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee = cp.GetCommittee(ctx, termChangeHeight+1)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")

			committee = cp.GetCommittee(ctx, termChangeHeight+2)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")
		})
	})
}

func TestManagement_GetCommitteeWhenTwoChangesClose(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			cp := newStaticCommitteeManagement(0, testKeys.NodeAddressesForTests()[:4])
			termChangeHeight := uint64(10)
			cp.addCommittee(termChangeHeight, testKeys.NodeAddressesForTests()[1:5])
			cp.addCommittee(termChangeHeight+2, testKeys.NodeAddressesForTests()[5:9])

			committee := cp.GetCommittee(ctx, termChangeHeight)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee = cp.GetCommittee(ctx, termChangeHeight+1)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee = cp.GetCommittee(ctx, termChangeHeight+2)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")

			committee = cp.GetCommittee(ctx, termChangeHeight+3)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")
		})
	})
}

func newStaticCommitteeManagement(ref uint64, committee []primitives.NodeAddress) *Service{
	return &Service{
		committees:[]*CommitteeTerm{{ref, committee}},
	}
}

func (s* Service) addCommittee(ref uint64, committee []primitives.NodeAddress) {
	s.Lock()
	defer s.Unlock()
	s.committees = append (s.committees, &CommitteeTerm{ref, committee})
}
