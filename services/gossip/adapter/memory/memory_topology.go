// Copyright 2020 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package memory

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/scribe/log"
	"sync"
)

type Config interface {
	GossipPeers() adapter.GossipPeers
}

type TopologyProvider struct {
	logger log.Logger
	sync.RWMutex
	topology adapter.GossipPeers
}

func NewTopologyProvider(config Config, logger log.Logger) *TopologyProvider {
	return  &TopologyProvider{topology: config.GossipPeers(), logger :logger}
}

func (tp *TopologyProvider) GetTopology(ctx context.Context) adapter.GossipPeers {
	tp.RLock()
	defer tp.RUnlock()
	return tp.topology
}
