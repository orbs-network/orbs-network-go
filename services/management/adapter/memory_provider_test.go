package adapter

import (
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestManagementMemory_AllowDoubleCommitteeOnSameBlock(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		cp := NewMemoryProvider([]primitives.NodeAddress{}, []*services.GossipPeer{}, harness.Logger)
		termChangeHeight := primitives.TimestampSeconds(10)
		err := cp.AddCommittee(termChangeHeight, testKeys.NodeAddressesForTests()[1:5])
		require.NoError(t, err)

		err = cp.AddCommittee(termChangeHeight-1, testKeys.NodeAddressesForTests()[1:5])
		require.Error(t, err, "must fail on smaller")

		err = cp.AddCommittee(termChangeHeight, testKeys.NodeAddressesForTests()[1:5])
		require.NoError(t, err, "must not fail on equal")
	})
}
