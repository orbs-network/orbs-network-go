package tcp

import "github.com/orbs-network/orbs-network-go/config"

type gossipPeers map[string]config.GossipPeer

func peerDiff(oldPeers gossipPeers, newPeers gossipPeers) (peersToRemove gossipPeers, peersToAdd gossipPeers) {
	peersToRemove = make(gossipPeers)
	peersToAdd = make(gossipPeers)

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

func peerHasChangedPortOrIPAddress(p1 config.GossipPeer, p2 config.GossipPeer) bool {
	return p1.GossipEndpoint() != p2.GossipEndpoint() || p1.GossipPort() != p2.GossipPort()
}
