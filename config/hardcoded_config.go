package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"time"
)

//TODO introduce FileSystemConfig

type identity struct {
	nodePublicKey  primitives.Ed25519PublicKey
	nodePrivateKey primitives.Ed25519PrivateKey
}

type consensusConfig struct {
	*identity
	federationNodes                              map[string]FederationNode
	constantConsensusLeader                      primitives.Ed25519PublicKey
	activeConsensusAlgo                          consensus.ConsensusAlgoType
	benchmarkConsensusRoundRetryIntervalMillisec uint32
}

type blockStorageConfig struct {
	blockSyncCommitTimeoutMillisec time.Duration
}

type consensusContextConfig struct {
	belowMinimalBlockDelayMillis uint32
	minimumTransactionsInBlock   int
}

type hardCodedFederationNode struct {
	nodePublicKey primitives.Ed25519PublicKey
}

type hardcodedConfig struct {
	*identity
	*consensusConfig
	*blockStorageConfig
	*consensusContextConfig
}

func NewHardCodedFederationNode(nodePublicKey primitives.Ed25519PublicKey) FederationNode {
	return &hardCodedFederationNode{
		nodePublicKey: nodePublicKey,
	}
}

func NewHardCodedConfig(
	federationNodes map[string]FederationNode,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
	benchmarkConsensusRoundRetryIntervalMillisec uint32,
	blockSyncCommitTimeoutMillisec uint32,
	belowMinimalBlockDelayMillis uint32,
	minimumTransactionsInBlock int,
) NodeConfig {

	return &hardcodedConfig{
		identity: &identity{
			nodePublicKey:  nodePublicKey,
			nodePrivateKey: nodePrivateKey,
		},
		consensusConfig: &consensusConfig{
			federationNodes:                              federationNodes,
			constantConsensusLeader:                      constantConsensusLeader,
			activeConsensusAlgo:                          activeConsensusAlgo,
			benchmarkConsensusRoundRetryIntervalMillisec: benchmarkConsensusRoundRetryIntervalMillisec,
		},
		blockStorageConfig: &blockStorageConfig{
			blockSyncCommitTimeoutMillisec: time.Duration(blockSyncCommitTimeoutMillisec) * time.Millisecond,
		},
		consensusContextConfig: &consensusContextConfig{
			belowMinimalBlockDelayMillis: belowMinimalBlockDelayMillis,
			minimumTransactionsInBlock:   minimumTransactionsInBlock,
		},
	}
}

func NewConsensusConfig(
	federationNodes map[string]FederationNode,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
	benchmarkConsensusRoundRetryIntervalMillisec uint32,
) *consensusConfig {

	return &consensusConfig{
		identity: &identity{
			nodePublicKey:  nodePublicKey,
			nodePrivateKey: nodePrivateKey,
		},
		federationNodes:                              federationNodes,
		constantConsensusLeader:                      constantConsensusLeader,
		activeConsensusAlgo:                          activeConsensusAlgo,
		benchmarkConsensusRoundRetryIntervalMillisec: benchmarkConsensusRoundRetryIntervalMillisec,
	}
}

func NewBlockStorageConfig(blockSyncCommitTimeoutMillisec uint32) *blockStorageConfig {
	return &blockStorageConfig{blockSyncCommitTimeoutMillisec: time.Duration(blockSyncCommitTimeoutMillisec) * time.Millisecond}
}

func NewConsensusContextConfig(belowMinimalBlockDelayMillis uint32, minimumTransactionsInBlock int) *consensusContextConfig {
	return &consensusContextConfig{
		belowMinimalBlockDelayMillis: belowMinimalBlockDelayMillis,
		minimumTransactionsInBlock:   minimumTransactionsInBlock,
	}
}

func (c *identity) NodePublicKey() primitives.Ed25519PublicKey {
	return c.nodePublicKey
}

func (c *identity) NodePrivateKey() primitives.Ed25519PrivateKey {
	return c.nodePrivateKey
}

func (c *consensusConfig) NetworkSize(asOfBlock uint64) uint32 {
	return uint32(len(c.federationNodes))
}

func (c *consensusConfig) FederationNodes(asOfBlock uint64) map[string]FederationNode {
	return c.federationNodes
}

func (c *consensusConfig) ConstantConsensusLeader() primitives.Ed25519PublicKey {
	return c.constantConsensusLeader
}

func (c *consensusConfig) ActiveConsensusAlgo() consensus.ConsensusAlgoType {
	return c.activeConsensusAlgo
}

func (c *consensusConfig) BenchmarkConsensusRoundRetryIntervalMillisec() uint32 {
	return c.benchmarkConsensusRoundRetryIntervalMillisec
}

func (n *hardCodedFederationNode) NodePublicKey() primitives.Ed25519PublicKey {
	return n.nodePublicKey
}

func (c *blockStorageConfig) BlockSyncCommitTimeoutMillisec() time.Duration {
	return c.blockSyncCommitTimeoutMillisec
}

func (c *consensusContextConfig) BelowMinimalBlockDelayMillis() uint32 {
	return c.belowMinimalBlockDelayMillis
}

func (c *consensusContextConfig) MinimumTransactionsInBlock() int {
	return c.minimumTransactionsInBlock
}
