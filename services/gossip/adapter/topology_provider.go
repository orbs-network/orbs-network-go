// Copyright 2020 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import "context"

type GossipPeer interface {
	GossipPort() int
	GossipEndpoint() string
	HexOrbsAddress() string
}

type GossipPeers map[string]GossipPeer

type gossipPeer struct {
	gossipPort     int
	gossipEndpoint string
	hexOrbsAddress string
}

func NewGossipPeer(gossipPort int, gossipEndpoint string, hexAddress string) GossipPeer {
	return &gossipPeer{
		gossipPort:     gossipPort,
		gossipEndpoint: gossipEndpoint,
		hexOrbsAddress: hexAddress,
	}
}

func (c *gossipPeer) GossipPort() int {
	return c.gossipPort
}

func (c *gossipPeer) GossipEndpoint() string {
	return c.gossipEndpoint
}

func (c *gossipPeer) HexOrbsAddress() string {
	return c.hexOrbsAddress
}

type TopologyProvider interface {
	GetTopology(ctx context.Context) GossipPeers
	UpdateTopology(ctx context.Context) error
}
