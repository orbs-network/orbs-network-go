package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
)

//TODO introduce FileSystemConfig

type hardcodedConfig struct {
	federationNodes                              map[string]FederationNode
	nodePublicKey                                primitives.Ed25519PublicKey
	nodePrivateKey                               primitives.Ed25519PrivateKey
	constantConsensusLeader                      primitives.Ed25519PublicKey
	activeConsensusAlgo                          consensus.ConsensusAlgoType
	benchmarkConsensusRoundRetryIntervalMillisec uint32
}

func NewHardCodedConfig(
	federationNodes map[string]FederationNode,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
	benchmarkConsensusRoundRetryIntervalMillisec uint32,
) NodeConfig {

	return &hardcodedConfig{
		federationNodes:                              federationNodes,
		nodePublicKey:                                nodePublicKey,
		nodePrivateKey:                               nodePrivateKey,
		constantConsensusLeader:                      constantConsensusLeader,
		activeConsensusAlgo:                          activeConsensusAlgo,
		benchmarkConsensusRoundRetryIntervalMillisec: benchmarkConsensusRoundRetryIntervalMillisec,
	}
}

func (c *hardcodedConfig) NetworkSize(asOfBlock uint64) uint32 {
	return uint32(len(c.federationNodes))
}

func (c *hardcodedConfig) FederationNodes(asOfBlock uint64) map[string]FederationNode {
	return c.federationNodes
}

func (c *hardcodedConfig) NodePublicKey() primitives.Ed25519PublicKey {
	return c.nodePublicKey
}

func (c *hardcodedConfig) NodePrivateKey() primitives.Ed25519PrivateKey {
	return c.nodePrivateKey
}

func (c *hardcodedConfig) ConstantConsensusLeader() primitives.Ed25519PublicKey {
	return c.constantConsensusLeader
}

func (c *hardcodedConfig) ActiveConsensusAlgo() consensus.ConsensusAlgoType {
	return c.activeConsensusAlgo
}

func (c *hardcodedConfig) BenchmarkConsensusRoundRetryIntervalMillisec() uint32 {
	return c.benchmarkConsensusRoundRetryIntervalMillisec
}

type hardCodedFederationNode struct {
	nodePublicKey primitives.Ed25519PublicKey
}

func NewHardCodedFederationNode(nodePublicKey primitives.Ed25519PublicKey) FederationNode {
	return &hardCodedFederationNode{
		nodePublicKey: nodePublicKey,
	}
}

func (n *hardCodedFederationNode) NodePublicKey() primitives.Ed25519PublicKey {
	return n.nodePublicKey
}
