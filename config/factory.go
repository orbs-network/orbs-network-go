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
	gossipListenPort uint16,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey) {
	c.SetNodePublicKey(nodePublicKey)
	c.SetNodePrivateKey(nodePrivateKey)
	c.SetUint32(GOSSIP_LISTEN_PORT, uint32(gossipListenPort))
}

func (c *config) MergeWithFileConfig(source string) (mutableNodeConfig, error) {
	return newFileConfig(c, source)
}
