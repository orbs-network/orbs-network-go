package tcp

import "github.com/orbs-network/orbs-network-go/config"

type gossipPeers map[string]config.GossipPeer

func peerDiff(oldPeers gossipPeers, newPeers gossipPeers) (peersToRemove gossipPeers, peersToAdd gossipPeers) {
	peersToRemove = make(gossipPeers)
	peersToAdd = make(gossipPeers)

	for a, n := range newPeers {
		if o, ok := oldPeers[a]; !ok || !peerEquals(n, o) {
			peersToAdd[a] = n
		}
	}

	for a, o := range oldPeers {
		if n, ok := newPeers[a]; !ok || !peerEquals(n, o) {
			peersToRemove[a] = o
		}
	}

	return
}

func peerEquals(p1 config.GossipPeer, p2 config.GossipPeer) bool {
	return p1.GossipEndpoint() == p2.GossipEndpoint() && p1.GossipPort() == p2.GossipPort()
}
