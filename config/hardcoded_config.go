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
	federationNodes                            map[string]FederationNode
	constantConsensusLeader                    primitives.Ed25519PublicKey
	activeConsensusAlgo                        consensus.ConsensusAlgoType
	benchmarkConsensusRoundRetryIntervalMillis uint32
}

type blockStorageConfig struct {
	blockSyncCommitTimeoutMillis time.Duration
}

type consensusContextConfig struct {
	belowMinimalBlockDelayMillis uint32
	minimumTransactionsInBlock   int
}

type stateStorageConfig struct {
	stateHistoryRetentionInBlockHeights uint64
}

type hardCodedFederationNode struct {
	nodePublicKey primitives.Ed25519PublicKey
}

type hardcodedConfig struct {
	*identity
	*consensusConfig
	*blockStorageConfig
	*stateStorageConfig
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
	benchmarkConsensusRoundRetryIntervalMillis uint32,
	blockSyncCommitTimeoutMillis uint32,
	stateHistoryRetentionInBlockHeights uint64,
	belowMinimalBlockDelayMillis uint32,
	minimumTransactionsInBlock int,
) NodeConfig {

	return &hardcodedConfig{
		identity: &identity{
			nodePublicKey:  nodePublicKey,
			nodePrivateKey: nodePrivateKey,
		},
		consensusConfig: &consensusConfig{
			federationNodes:                            federationNodes,
			constantConsensusLeader:                    constantConsensusLeader,
			activeConsensusAlgo:                        activeConsensusAlgo,
			benchmarkConsensusRoundRetryIntervalMillis: benchmarkConsensusRoundRetryIntervalMillis,
		},
		blockStorageConfig: &blockStorageConfig{
			blockSyncCommitTimeoutMillis: time.Duration(blockSyncCommitTimeoutMillis) * time.Millisecond,
		},
		stateStorageConfig: &stateStorageConfig{stateHistoryRetentionInBlockHeights: stateHistoryRetentionInBlockHeights},
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
	benchmarkConsensusRoundRetryIntervalMillis uint32,
) *consensusConfig {

	return &consensusConfig{
		identity: &identity{
			nodePublicKey:                                nodePublicKey,
			nodePrivateKey:                               nodePrivateKey,
		},
		federationNodes:                            federationNodes,
		constantConsensusLeader:                    constantConsensusLeader,
		activeConsensusAlgo:                        activeConsensusAlgo,
		benchmarkConsensusRoundRetryIntervalMillis: benchmarkConsensusRoundRetryIntervalMillis,
	}
}

func NewBlockStorageConfig(blockSyncCommitTimeoutMillisec uint32) *blockStorageConfig {
	return &blockStorageConfig{blockSyncCommitTimeoutMillis: time.Duration(blockSyncCommitTimeoutMillisec) * time.Millisecond}
}

func NewConsensusContextConfig(belowMinimalBlockDelayMillis uint32, minimumTransactionsInBlock int) *consensusContextConfig {
	return &consensusContextConfig{
		belowMinimalBlockDelayMillis: belowMinimalBlockDelayMillis,
		minimumTransactionsInBlock:   minimumTransactionsInBlock,
	}
}

func NewStateStorageConfig(maxStateHistory uint64) *stateStorageConfig {
	return &stateStorageConfig{stateHistoryRetentionInBlockHeights: maxStateHistory}
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

func (c *consensusConfig) BenchmarkConsensusRoundRetryIntervalMillis() uint32 {
	return c.benchmarkConsensusRoundRetryIntervalMillis
}

func (n *hardCodedFederationNode) NodePublicKey() primitives.Ed25519PublicKey {
	return n.nodePublicKey
}

func (c *blockStorageConfig) BlockSyncCommitTimeoutMillis() time.Duration {
	return c.blockSyncCommitTimeoutMillis
}

func (c *consensusContextConfig) BelowMinimalBlockDelayMillis() uint32 {
	return c.belowMinimalBlockDelayMillis
}

func (c *consensusContextConfig) MinimumTransactionsInBlock() int {
	return c.minimumTransactionsInBlock
}

func (c *stateStorageConfig) StateHistoryRetentionInBlockHeights() uint64 {
	return c.stateHistoryRetentionInBlockHeights
}
