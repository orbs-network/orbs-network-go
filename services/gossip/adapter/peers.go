// Copyright 2020 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

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

func PeerDiff(oldPeers GossipPeers, newPeers GossipPeers) (peersToRemove GossipPeers, peersToAdd GossipPeers) {
	peersToRemove = make(GossipPeers)
	peersToAdd = make(GossipPeers)

	for a, n := range newPeers {
		if o, peerExistsInOldList := oldPeers[a]; !peerExistsInOldList || peerHasChangedPortOrIPAddress(n, o) {
			peersToAdd[a] = n
		}
	}

	for a, o := range oldPeers {
		if n, peerExistsInNewList := newPeers[a]; !peerExistsInNewList || peerHasChangedPortOrIPAddress(n, o) {
			peersToRemove[a] = o
		}
	}

	return
}

func peerHasChangedPortOrIPAddress(p1 GossipPeer, p2 GossipPeer) bool {
	return p1.GossipEndpoint() != p2.GossipEndpoint() || p1.GossipPort() != p2.GossipPort()
}
