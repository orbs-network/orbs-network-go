package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

func emptyConfig() mutableNodeConfig {
	return &config{
		kv: make(map[string]NodeConfigValue),
	}
}
func (c *config) OverrideNodeSpecificValues(
	gossipListenPort int,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey) NodeConfig {
	cloned := c.Clone()
	cloned.SetNodePublicKey(nodePublicKey)
	cloned.SetNodePrivateKey(nodePrivateKey)
	cloned.SetUint32(GOSSIP_LISTEN_PORT, uint32(gossipListenPort))

	return cloned
}

func (c *config) MergeWithFileConfig(source string) (mutableNodeConfig, error) {
	return newFileConfig(c, source)
}

func (c *config) Clone() mutableNodeConfig {
	return &config{
		activeConsensusAlgo:     c.activeConsensusAlgo,
		constantConsensusLeader: c.constantConsensusLeader,
		federationNodes:         c.federationNodes,
		gossipPeers:             c.gossipPeers,
		nodePrivateKey:          c.nodePrivateKey,
		nodePublicKey:           c.nodePublicKey,
		kv:                      cloneMap(c.kv),
	}
}

func cloneMap(kv map[string]NodeConfigValue) (cloned map[string]NodeConfigValue) {
	cloned = make(map[string]NodeConfigValue)
	for k, v := range kv {
		cloned[k] = v
	}
	return cloned
}
