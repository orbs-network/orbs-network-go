package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
)

func emptyConfig() mutableNodeConfig {
	return &config{
		kv: make(map[string]NodeConfigValue),
	}
}
func (c *config) OverrideNodeSpecificValues(
	federationNodes map[string]FederationNode,
	gossipPeers map[string]GossipPeer,
	gossipListenPort uint16,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType) {
	c.SetFederationNodes(federationNodes)
	c.SetGossipPeers(gossipPeers)
	c.SetNodePublicKey(nodePublicKey)
	c.SetNodePrivateKey(nodePrivateKey)
	c.SetConstantConsensusLeader(constantConsensusLeader)
	c.SetActiveConsensusAlgo(activeConsensusAlgo)
	c.SetUint32(GOSSIP_LISTEN_PORT, uint32(gossipListenPort))
}

func (c *config) MergeWithFileConfig(source string) (mutableNodeConfig, error) {
	return NewFileConfig(c, source)
}
