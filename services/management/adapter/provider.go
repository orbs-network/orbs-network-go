package adapter

import (
	"context"
	adapterGossipPeers "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/management"
)

type Provider interface {
	Update(ctx context.Context) (uint64, adapterGossipPeers.GossipPeers, []*management.CommitteeTerm, error)
}
