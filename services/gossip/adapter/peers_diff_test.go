package adapter

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPeerDiff_TwoEmptyPeerLists_ReturnEmptyResults(t *testing.T) {
	toRemove, toAdd := PeerDiff(make(GossipPeers), make(GossipPeers))
	require.Empty(t, toRemove)
	require.Empty(t, toAdd)
}

func TestPeerDiff_OldIsEmpty_ReturnsEmptyToRemove_AndAPeerToAdd(t *testing.T) {
	oldPeers := make(GossipPeers)
	newPeers := make(GossipPeers)

	newPeers["1"] = NewGossipPeer(1, "10.0.0.1", "")

	toRemove, toAdd := PeerDiff(oldPeers, newPeers)

	require.Empty(t, toRemove, "no peers should be removed")

	peer, ok := toAdd["1"]
	require.True(t, ok, "an added peer was missing from peersToAdd")
	require.Equal(t, peer, newPeers["1"], "an added peer was missing from toAdd")
}

func TestPeerDiff_OldHasAPeer_ReturnsPeerToRemove(t *testing.T) {
	oldPeers := make(GossipPeers)
	newPeers := make(GossipPeers)

	oldPeers["1"] = NewGossipPeer(1, "10.0.0.1", "")

	toRemove, toAdd := PeerDiff(oldPeers, newPeers)

	peer, ok := toRemove["1"]
	require.True(t, ok, "a removed peer was missing from peersToRemove")
	require.Equal(t, peer, oldPeers["1"], "a removed peer was missing from toRemove")

	require.Empty(t, toAdd, "No peers should be added")
}

func TestPeerDiff_ReturnsEmptyToAddAndToRemoveLists_WhenConfigIsNotChanged(t *testing.T) {
	peers := make(GossipPeers)
	peers["1"] = NewGossipPeer(1, "10.0.0.1", "")

	toRemove, toAdd := PeerDiff(peers, peers)

	require.Empty(t, toRemove)
	require.Empty(t, toAdd)
}

func TestPeerDiff_Returns_CorrectLists_WhenAPeerWasAddedAndAnotherWasRemoved(t *testing.T) {
	oldPeers := make(GossipPeers)
	newPeers := make(GossipPeers)

	oldPeers["1"] = NewGossipPeer(1, "10.0.0.1", "")
	oldPeers["2"] = NewGossipPeer(2, "10.0.0.2", "")

	newPeers["2"] = NewGossipPeer(2, "10.0.0.2", "")
	newPeers["3"] = NewGossipPeer(3, "10.0.0.3", "")

	toRemove, toAdd := PeerDiff(oldPeers, newPeers)

	peer, ok := toRemove["1"]
	require.True(t, ok, "a removed peer was missing from peersToRemove")
	require.Equal(t, peer, oldPeers["1"], "a removed peer was missing from toRemove")
	require.Len(t, toRemove, 1, "Expected toRemove to contain exactly 1 element")

	addedPeer, ok := toAdd["3"]
	require.True(t, ok, "an added peer was missing from peersToAdd")
	require.Equal(t, addedPeer, newPeers["3"], "an added peer was missing from toAdd")
	require.Len(t, toAdd, 1, "Expected toAdd to contain exactly 1 element")
}

func TestPeerDiff_Returns_PeersThatChangedAddress_InBothLists(t *testing.T) {
	oldPeers := make(GossipPeers)
	newPeers := make(GossipPeers)

	oldPeers["1"] = NewGossipPeer(1, "10.0.0.1", "")
	newPeers["1"] = NewGossipPeer(3, "10.0.0.1", "")
	oldPeers["2"] = NewGossipPeer(1, "10.0.0.2", "")
	newPeers["2"] = NewGossipPeer(1, "10.0.0.3", "")

	toRemove, toAdd := PeerDiff(oldPeers, newPeers)

	require.Len(t, toAdd, 2, "Expected toAdd to contain both peers")
	require.Len(t, toRemove, 2, "Expected toAdd to contain both peers")
}
