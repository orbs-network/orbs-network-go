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
	nodeAddress primitives.NodeAddress,
	nodePrivateKey primitives.EcdsaSecp256K1PrivateKey) NodeConfig {
	cloned := c.Clone()
	cloned.SetNodeAddress(nodeAddress)
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
		nodeAddress:             c.nodeAddress,
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
