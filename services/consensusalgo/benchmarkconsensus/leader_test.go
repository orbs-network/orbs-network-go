package benchmarkconsensus

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLeaderQuorum(t *testing.T) {
	nodes := make(map[string]config.FederationNode)

	for i := 0; i < 6; i++ {
		nodes[fmt.Sprintf("fake-key-node%d", i)] = nil
	}

	cfg := config.ForProduction("")
	cfg.SetFederationNodes(nodes)

	require.NotZero(t, cfg.NetworkSize(0))

	s := &service{
		config: cfg,
	}

	require.NotZero(t, s.requiredQuorumSize())
}
