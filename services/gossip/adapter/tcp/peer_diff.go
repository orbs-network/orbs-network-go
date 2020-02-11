package tcp

import (
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
)

func peerDiff(oldPeers adapter.GossipPeers, newPeers adapter.GossipPeers) (peersToRemove adapter.GossipPeers, peersToAdd adapter.GossipPeers) {
	peersToRemove = make(adapter.GossipPeers)
	peersToAdd = make(adapter.GossipPeers)

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

func peerHasChangedPortOrIPAddress(p1 adapter.GossipPeer, p2 adapter.GossipPeer) bool {
	return p1.GossipEndpoint() != p2.GossipEndpoint() || p1.GossipPort() != p2.GossipPort()
}
