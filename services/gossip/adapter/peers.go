// Copyright 2020 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type TransportPeer interface {
	Port() int
	Endpoint() string
	HexOrbsAddress() string
}

type TransportPeers map[string]TransportPeer

type peer struct {
	port           int
	endpoint       string
	hexOrbsAddress string
}

func NewGossipPeers(servicePeers []*services.GossipPeer) TransportPeers {
	peers := make(TransportPeers, len(servicePeers))
	for _, peer := range servicePeers {
		peers[peer.Address.KeyForMap()] = NewGossipPeer(int(peer.Port), peer.Endpoint, hex.EncodeToString(peer.Address))
	}
	return peers
}

func NewGossipPeer(gossipPort int, gossipEndpoint string, hexAddress string) TransportPeer {
	return &peer{
		port:           gossipPort,
		endpoint:       gossipEndpoint,
		hexOrbsAddress: hexAddress,
	}
}

func (c *peer) Port() int {
	return c.port
}

func (c *peer) Endpoint() string {
	return c.endpoint
}

func (c *peer) HexOrbsAddress() string {
	return c.hexOrbsAddress
}

func PeerDiff(oldPeers TransportPeers, newPeers TransportPeers) (peersToRemove TransportPeers, peersToAdd TransportPeers) {
	peersToRemove = make(TransportPeers)
	peersToAdd = make(TransportPeers)

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

func peerHasChangedPortOrIPAddress(p1 TransportPeer, p2 TransportPeer) bool {
	return p1.Endpoint() != p2.Endpoint() || p1.Port() != p2.Port()
}
