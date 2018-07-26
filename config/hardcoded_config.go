package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
)

//TODO introduce FileSystemConfig
type hardcodedConfig struct {
	networkSize             uint32
	nodePublicKey           primitives.Ed25519PublicKey
	constantConsensusLeader primitives.Ed25519PublicKey
	activeConsensusAlgo     consensus.ConsensusAlgoType
}

func NewHardCodedConfig(
	networkSize uint32,
	nodePublicKey primitives.Ed25519PublicKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
) NodeConfig {

	return &hardcodedConfig{
		networkSize:             networkSize,
		nodePublicKey:           nodePublicKey,
		constantConsensusLeader: constantConsensusLeader,
		activeConsensusAlgo:     activeConsensusAlgo,
	}
}

func (c *hardcodedConfig) NetworkSize(asOfBlock uint64) uint32 {
	return c.networkSize
}

func (c *hardcodedConfig) NodePublicKey() primitives.Ed25519PublicKey {
	return c.nodePublicKey
}

func (c *hardcodedConfig) ConstantConsensusLeader() primitives.Ed25519PublicKey {
	return c.constantConsensusLeader
}

func (c *hardcodedConfig) ActiveConsensusAlgo() consensus.ConsensusAlgoType {
	return c.activeConsensusAlgo
}
