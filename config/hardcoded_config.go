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
	virtualChainId primitives.VirtualChainId
}

type consensusConfig struct {
	*identity
	federationNodes                            map[string]FederationNode
	constantConsensusLeader                    primitives.Ed25519PublicKey
	activeConsensusAlgo                        consensus.ConsensusAlgoType
	benchmarkConsensusRoundRetryIntervalMillis uint32
}

type crossServiceConfig struct {
	queryGraceTimeoutMillis uint64
	querySyncGraceBlockDist uint16
}

type blockStorageConfig struct {
	blockSyncCommitTimeoutMillis                     time.Duration
	blockTransactionReceiptQueryStartGraceSec        time.Duration
	blockTransactionReceiptQueryEndGraceSec          time.Duration
	blockTransactionReceiptQueryTransactionExpireSec time.Duration
}

type consensusContextConfig struct {
	belowMinimalBlockDelayMillis uint32
	minimumTransactionsInBlock   int
}

type stateStorageConfig struct {
	*crossServiceConfig
	stateHistoryRetentionInBlockHeights uint16
}

type transactionPoolConfig struct {
	*identity
	*crossServiceConfig
	pendingPoolSizeInBytes               uint32
	transactionExpirationWindowInSeconds uint32
	futureTimestampGraceInSeconds        uint32
}

type hardCodedFederationNode struct {
	nodePublicKey primitives.Ed25519PublicKey
}

type hardcodedConfig struct {
	*identity
	*consensusConfig
	*crossServiceConfig
	*blockStorageConfig
	*stateStorageConfig
	*consensusContextConfig
	*transactionPoolConfig
}

func NewHardCodedFederationNode(nodePublicKey primitives.Ed25519PublicKey) FederationNode {
	return &hardCodedFederationNode{
		nodePublicKey: nodePublicKey,
	}
}

func ForProduction(
	federationNodes map[string]FederationNode,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
	benchmarkConsensusRoundRetryIntervalMillis uint32,
	minimumTransactionsInBlock int,
) NodeConfig {

	return newHardCodedConfig(
		federationNodes,
		nodePublicKey,
		nodePrivateKey,
		constantConsensusLeader,
		activeConsensusAlgo,
		benchmarkConsensusRoundRetryIntervalMillis,
		minimumTransactionsInBlock,
		20) // longer than in acceptance test because otherwise e2e flakes. TODO figure out why

}

func ForAcceptanceTests(
	federationNodes map[string]FederationNode,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
) NodeConfig {

	return newHardCodedConfig(
		federationNodes,
		nodePublicKey,
		nodePrivateKey,
		constantConsensusLeader,
		activeConsensusAlgo,
		1,
		1,
		1)
}

func newHardCodedConfig(
	federationNodes map[string]FederationNode,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
	benchmarkConsensusRoundRetryIntervalMillis uint32,
	minimumTransactionsInBlock int,
	belowMinimalBlockDelayMillis uint32,
) NodeConfig {

	return &hardcodedConfig{
		identity: &identity{
			nodePublicKey:  nodePublicKey,
			nodePrivateKey: nodePrivateKey,
			virtualChainId: 42,
		},
		consensusConfig: &consensusConfig{
			federationNodes:                            federationNodes,
			constantConsensusLeader:                    constantConsensusLeader,
			activeConsensusAlgo:                        activeConsensusAlgo,
			benchmarkConsensusRoundRetryIntervalMillis: benchmarkConsensusRoundRetryIntervalMillis,
		},
		crossServiceConfig: &crossServiceConfig{
			queryGraceTimeoutMillis: 100,
			querySyncGraceBlockDist: 3,
		},
		blockStorageConfig: &blockStorageConfig{
			blockSyncCommitTimeoutMillis:                     70 * time.Millisecond,
			blockTransactionReceiptQueryStartGraceSec:        5 * time.Second,
			blockTransactionReceiptQueryEndGraceSec:          5 * time.Second,
			blockTransactionReceiptQueryTransactionExpireSec: 180 * time.Second,
		},
		stateStorageConfig: &stateStorageConfig{
			stateHistoryRetentionInBlockHeights: 5,
		},
		consensusContextConfig: &consensusContextConfig{
			belowMinimalBlockDelayMillis: belowMinimalBlockDelayMillis,
			minimumTransactionsInBlock:   minimumTransactionsInBlock,
		},
		transactionPoolConfig: &transactionPoolConfig{
			pendingPoolSizeInBytes:               20 * 1024 * 1024,
			transactionExpirationWindowInSeconds: 1800,
			futureTimestampGraceInSeconds:        180,
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
			nodePublicKey:  nodePublicKey,
			nodePrivateKey: nodePrivateKey,
			virtualChainId: 42,
		},
		federationNodes:                            federationNodes,
		constantConsensusLeader:                    constantConsensusLeader,
		activeConsensusAlgo:                        activeConsensusAlgo,
		benchmarkConsensusRoundRetryIntervalMillis: benchmarkConsensusRoundRetryIntervalMillis,
	}
}

func NewBlockStorageConfig(blockSyncCommitTimeoutMillis, blockTransactionReceiptQueryStartGraceSec, blockTransactionReceiptQueryEndGraceSec, blockTransactionReceiptQueryTransactionExpireSec uint32) *blockStorageConfig {
	return &blockStorageConfig{
		blockSyncCommitTimeoutMillis:                     time.Duration(blockSyncCommitTimeoutMillis) * time.Millisecond,
		blockTransactionReceiptQueryStartGraceSec:        time.Duration(blockTransactionReceiptQueryStartGraceSec) * time.Second,
		blockTransactionReceiptQueryEndGraceSec:          time.Duration(blockTransactionReceiptQueryEndGraceSec) * time.Second,
		blockTransactionReceiptQueryTransactionExpireSec: time.Duration(blockTransactionReceiptQueryTransactionExpireSec) * time.Second,
	}
}

func NewConsensusContextConfig(belowMinimalBlockDelayMillis uint32, minimumTransactionsInBlock int) *consensusContextConfig {
	return &consensusContextConfig{
		belowMinimalBlockDelayMillis: belowMinimalBlockDelayMillis,
		minimumTransactionsInBlock:   minimumTransactionsInBlock,
	}
}

func NewTransactionPoolConfig(pendingPoolSizeInBytes uint32, transactionExpirationWindowInSeconds uint32, nodePublicKey primitives.Ed25519PublicKey) *transactionPoolConfig {
	return &transactionPoolConfig{
		identity: &identity{
			nodePublicKey:  nodePublicKey,
			virtualChainId: 42,
		},
		crossServiceConfig: &crossServiceConfig{
			queryGraceTimeoutMillis: 100,
			querySyncGraceBlockDist: 5,
		},
		pendingPoolSizeInBytes:               pendingPoolSizeInBytes,
		transactionExpirationWindowInSeconds: transactionExpirationWindowInSeconds,
		futureTimestampGraceInSeconds:        180,
	}
}

func NewStateStorageConfig(maxStateHistory uint16, graceBlockDist uint16, graceTimeoutMillis uint64) *stateStorageConfig {
	return &stateStorageConfig{
		stateHistoryRetentionInBlockHeights: maxStateHistory,
		crossServiceConfig: &crossServiceConfig{
			queryGraceTimeoutMillis: graceTimeoutMillis,
			querySyncGraceBlockDist: graceBlockDist,
		},
	}
}

func (c *identity) NodePublicKey() primitives.Ed25519PublicKey {
	return c.nodePublicKey
}

func (c *identity) NodePrivateKey() primitives.Ed25519PrivateKey {
	return c.nodePrivateKey
}

func (c *identity) VirtualChainId() primitives.VirtualChainId {
	return c.virtualChainId
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

func (c *blockStorageConfig) BlockTransactionReceiptQueryStartGraceSec() time.Duration {
	return c.blockTransactionReceiptQueryStartGraceSec
}
func (c *blockStorageConfig) BlockTransactionReceiptQueryEndGraceSec() time.Duration {
	return c.blockTransactionReceiptQueryEndGraceSec
}
func (c *blockStorageConfig) BlockTransactionReceiptQueryTransactionExpireSec() time.Duration {
	return c.blockTransactionReceiptQueryTransactionExpireSec
}

func (c *consensusContextConfig) BelowMinimalBlockDelayMillis() uint32 {
	return c.belowMinimalBlockDelayMillis
}

func (c *consensusContextConfig) MinimumTransactionsInBlock() int {
	return c.minimumTransactionsInBlock
}

func (c *stateStorageConfig) StateHistoryRetentionInBlockHeights() uint16 {
	return c.stateHistoryRetentionInBlockHeights
}

func (c *crossServiceConfig) QuerySyncGraceBlockDist() uint16 {
	return c.querySyncGraceBlockDist
}

func (c *crossServiceConfig) QueryGraceTimeoutMillis() uint64 {
	return c.queryGraceTimeoutMillis
}

func (c *transactionPoolConfig) PendingPoolSizeInBytes() uint32 {
	return c.pendingPoolSizeInBytes
}

func (c *transactionPoolConfig) TransactionExpirationWindowInSeconds() uint32 {
	return c.transactionExpirationWindowInSeconds
}

func (c *transactionPoolConfig) FutureTimestampGraceInSeconds() uint32 {
	return c.futureTimestampGraceInSeconds
}
