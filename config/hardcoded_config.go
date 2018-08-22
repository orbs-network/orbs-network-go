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

type configValue struct {
	StringValue   string
	BytesValue    []byte
	Uint32Value   uint32
	DurationValue time.Duration
}

type config struct {
	kv                      map[string]configValue
	federationNodes         map[string]FederationNode
	nodePublicKey           primitives.Ed25519PublicKey
	nodePrivateKey          primitives.Ed25519PrivateKey
	constantConsensusLeader primitives.Ed25519PublicKey
	activeConsensusAlgo     consensus.ConsensusAlgoType
}

const (
	VIRTUAL_CHAIN_ID                          = "VIRTUAL_CHAIN_ID"
	BENCHMARK_CONSENSUS_RETRY_INTERVAL_MILLIS = "BENCHMARK_CONSENSUS_RETRY_INTERVAL_MILLIS"

	BLOCK_SYNC_COMMIT_TIMEOUT_MILLIS                       = "BLOCK_SYNC_COMMIT_TIMEOUT_MILLIS"
	BLOCK_TRANSACTION_RECEIPT_QUERY_START_GRACE_SEC        = "BLOCK_TRANSACTION_RECEIPT_QUERY_START_GRACE_SEC"
	BLOCK_TRANSACTION_RECEIPT_QUERY_END_GRACE_SEC          = "BLOCK_TRANSACTION_RECEIPT_QUERY_END_GRACE_SEC"
	BLOCK_TRANSACTION_RECEIPT_QUERY_TRANSACTION_EXPIRE_SEC = "BLOCK_TRANSACTION_RECEIPT_QUERY_TRANSACTION_EXPIRE_SEC"

	BELOW_MINIMAL_BLOCK_DELAY_MILLIS         = "BELOW_MINIMAL_BLOCK_DELAY_MILLIS"
	MINIMUM_TRANSACTION_IN_BLOCK             = "MINIMUM_TRANSACTION_IN_BLOCK"
	STATE_HISTORY_RETENTION_IN_BLOCK_HEIGHTS = "STATE_HISTORY_RETENTION_IN_BLOCK_HEIGHTS"

	QUERY_SYNC_GRACE_BLOCK_DIST = "QUERY_SYNC_GRACE_BLOCK_DIST"
	QUERY_GRACE_TIMEOUT_MILLIS  = "QUERY_GRACE_TIMEOUT_MILLIS"

	PENDING_POOL_SIZE_IN_BYTES               = "PENDING_POOL_SIZE_IN_BYTES"
	TRANSACTION_EXPIRATION_WINDOW_IN_SECONDS = "TRANSACTION_EXPIRATION_WINDOW_IN_SECONDS"
	FUTURE_TIMESTAMP_GRACE_IN_SECONDS        = "FUTURE_TIMESTAMP_GRACE_IN_SECONDS"
)

func NewHardCodedFederationNode(nodePublicKey primitives.Ed25519PublicKey) FederationNode {
	return &hardCodedFederationNode{
		nodePublicKey: nodePublicKey,
	}
	return nil
}

func newHardCodedConfig(
	federationNodes map[string]FederationNode,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
	benchmarkConsensusRoundRetryIntervalMillis uint32,
	minimumTransactionsInBlock uint32,
	belowMinimalBlockDelayMillis uint32,
	queryGraceTimeoutMillis uint64,
) NodeConfig {
	cfg := &config{
		federationNodes:         federationNodes,
		nodePublicKey:           nodePublicKey,
		nodePrivateKey:          nodePrivateKey,
		constantConsensusLeader: constantConsensusLeader,
		activeConsensusAlgo:     activeConsensusAlgo,
		kv:                      make(map[string]configValue),
	}

	cfg.kv[VIRTUAL_CHAIN_ID] = configValue{Uint32Value: 42}
	cfg.kv[BENCHMARK_CONSENSUS_RETRY_INTERVAL_MILLIS] = configValue{Uint32Value: benchmarkConsensusRoundRetryIntervalMillis}

	cfg.kv[QUERY_GRACE_TIMEOUT_MILLIS] = configValue{DurationValue: time.Duration(queryGraceTimeoutMillis) * time.Millisecond}
	cfg.kv[QUERY_SYNC_GRACE_BLOCK_DIST] = configValue{Uint32Value: 3}

	cfg.kv[BLOCK_SYNC_COMMIT_TIMEOUT_MILLIS] = configValue{DurationValue: 70 * time.Millisecond}
	cfg.kv[BLOCK_TRANSACTION_RECEIPT_QUERY_START_GRACE_SEC] = configValue{DurationValue: 5 * time.Second}
	cfg.kv[BLOCK_TRANSACTION_RECEIPT_QUERY_END_GRACE_SEC] = configValue{DurationValue: 5 * time.Second}
	cfg.kv[BLOCK_TRANSACTION_RECEIPT_QUERY_TRANSACTION_EXPIRE_SEC] = configValue{DurationValue: 180 * time.Second}

	cfg.kv[STATE_HISTORY_RETENTION_IN_BLOCK_HEIGHTS] = configValue{Uint32Value: 5}

	cfg.kv[BELOW_MINIMAL_BLOCK_DELAY_MILLIS] = configValue{Uint32Value: belowMinimalBlockDelayMillis}
	cfg.kv[MINIMUM_TRANSACTION_IN_BLOCK] = configValue{Uint32Value: minimumTransactionsInBlock}

	cfg.kv[STATE_HISTORY_RETENTION_IN_BLOCK_HEIGHTS] = configValue{Uint32Value: 5}

	cfg.kv[PENDING_POOL_SIZE_IN_BYTES] = configValue{Uint32Value: 20 * 1024 * 1024}
	cfg.kv[TRANSACTION_EXPIRATION_WINDOW_IN_SECONDS] = configValue{Uint32Value: 1800}
	cfg.kv[FUTURE_TIMESTAMP_GRACE_IN_SECONDS] = configValue{Uint32Value: 180}

	return cfg
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

func (c *hardCodedFederationNode) NodePublicKey() primitives.Ed25519PublicKey {
	return c.nodePublicKey
}

func (c *config) NodePublicKey() primitives.Ed25519PublicKey {
	return c.nodePublicKey
}

func (c *config) NodePrivateKey() primitives.Ed25519PrivateKey {
	return c.nodePrivateKey
}

func (c *config) VirtualChainId() primitives.VirtualChainId {
	return primitives.VirtualChainId(c.kv[VIRTUAL_CHAIN_ID].Uint32Value)
}

func (c *config) NetworkSize(asOfBlock uint64) uint32 {
	return uint32(len(c.federationNodes))
}

func (c *config) FederationNodes(asOfBlock uint64) map[string]FederationNode {
	return c.federationNodes
}

func (c *config) ConstantConsensusLeader() primitives.Ed25519PublicKey {
	return c.constantConsensusLeader
}

func (c *config) ActiveConsensusAlgo() consensus.ConsensusAlgoType {
	return c.activeConsensusAlgo
}

func (c *config) BenchmarkConsensusRoundRetryIntervalMillis() uint32 {
	return c.kv[BENCHMARK_CONSENSUS_RETRY_INTERVAL_MILLIS].Uint32Value

}

func (c *config) BlockSyncCommitTimeoutMillis() time.Duration {
	return c.kv[BLOCK_SYNC_COMMIT_TIMEOUT_MILLIS].DurationValue
}

func (c *config) BlockTransactionReceiptQueryStartGraceSec() time.Duration {
	return c.kv[BLOCK_TRANSACTION_RECEIPT_QUERY_START_GRACE_SEC].DurationValue
}
func (c *config) BlockTransactionReceiptQueryEndGraceSec() time.Duration {
	return c.kv[BLOCK_TRANSACTION_RECEIPT_QUERY_END_GRACE_SEC].DurationValue
}
func (c *config) BlockTransactionReceiptQueryTransactionExpireSec() time.Duration {
	return c.kv[BLOCK_TRANSACTION_RECEIPT_QUERY_TRANSACTION_EXPIRE_SEC].DurationValue
}

func (c *config) BelowMinimalBlockDelayMillis() uint32 {
	return c.kv[BELOW_MINIMAL_BLOCK_DELAY_MILLIS].Uint32Value
}

func (c *config) MinimumTransactionsInBlock() uint32 {
	return c.kv[MINIMUM_TRANSACTION_IN_BLOCK].Uint32Value
}

func (c *config) StateHistoryRetentionInBlockHeights() uint32 {
	return c.kv[STATE_HISTORY_RETENTION_IN_BLOCK_HEIGHTS].Uint32Value
}

func (c *config) QuerySyncGraceBlockDist() uint32 {
	return c.kv[QUERY_SYNC_GRACE_BLOCK_DIST].Uint32Value

}

func (c *config) QueryGraceTimeoutMillis() time.Duration {
	return c.kv[QUERY_GRACE_TIMEOUT_MILLIS].DurationValue
}

func (c *config) PendingPoolSizeInBytes() uint32 {
	return c.kv[PENDING_POOL_SIZE_IN_BYTES].Uint32Value
}

func (c *config) TransactionExpirationWindowInSeconds() uint32 {
	return c.kv[TRANSACTION_EXPIRATION_WINDOW_IN_SECONDS].Uint32Value
}

func (c *config) FutureTimestampGraceInSeconds() uint32 {
	return c.kv[FUTURE_TIMESTAMP_GRACE_IN_SECONDS].Uint32Value
}
