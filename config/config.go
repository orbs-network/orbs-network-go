package config

import "github.com/orbs-network/orbs-network-go/consensus"

type NodeConfig interface {
	consensus.Config
}

//TODO introduce FileSystemConfig
type hardcodedConfig struct {
	networkSize uint32
}

func NewHardCodedConfig(networkSize uint32) NodeConfig {
	return &hardcodedConfig{networkSize: networkSize}
}

func (c *hardcodedConfig) GetNetworkSize(asOfBlock uint64) uint32 {
	return c.networkSize
}
