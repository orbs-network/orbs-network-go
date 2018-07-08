package config

type NodeConfig interface {
	NodeId() string
	NetworkSize(asOfBlock uint64) uint32
}

//TODO introduce FileSystemConfig
type hardcodedConfig struct {
	networkSize uint32
	nodeId string
}

func NewHardCodedConfig(networkSize uint32, nodeId string) NodeConfig {
	return &hardcodedConfig{networkSize: networkSize, nodeId: nodeId}
}

func (c *hardcodedConfig) NetworkSize(asOfBlock uint64) uint32 {
	return c.networkSize
}

func (c *hardcodedConfig) NodeId() string {
	return c.nodeId
}