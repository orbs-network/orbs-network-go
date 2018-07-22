package config

import "github.com/orbs-network/orbs-spec/types/go/primitives"

//TODO introduce FileSystemConfig
type hardcodedConfig struct {
	networkSize   uint32
	nodePublicKey primitives.Ed25519Pkey
}

func NewHardCodedConfig(networkSize uint32, nodePublicKey primitives.Ed25519Pkey) NodeConfig {
	return &hardcodedConfig{networkSize: networkSize, nodePublicKey: nodePublicKey}
}

func (c *hardcodedConfig) NetworkSize(asOfBlock uint64) uint32 {
	return c.networkSize
}

func (c *hardcodedConfig) NodePublicKey() primitives.Ed25519Pkey {
	return c.nodePublicKey
}
