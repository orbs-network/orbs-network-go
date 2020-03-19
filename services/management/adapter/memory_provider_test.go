package adapter

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestManagementMemory_PreventDoubleCommitteeOnSameBlock(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		cp := NewMemoryProvider(newMemoryConfig(), harness.Logger)
		termChangeHeight := uint64(10)
		err := cp.AddCommittee(termChangeHeight, testKeys.NodeAddressesForTests()[1:5])
		require.NoError(t, err)

		err = cp.AddCommittee(termChangeHeight-1, testKeys.NodeAddressesForTests()[1:5])
		require.Error(t, err, "must fail on smaller")

		err = cp.AddCommittee(termChangeHeight, testKeys.NodeAddressesForTests()[1:5])
		require.Error(t, err, "must fail on equal")
	})
}

type cfg struct {
}

func newMemoryConfig() *cfg {
	return &cfg{}
}

func (c *cfg) GossipPeers() adapter.GossipPeers {
	return make(adapter.GossipPeers)
}
func (c *cfg) GenesisValidatorNodes() map[string]config.ValidatorNode {
	return make(map[string]config.ValidatorNode)
}

