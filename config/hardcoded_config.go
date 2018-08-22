package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"time"
)

type hardCodedFederationNode struct {
	nodePublicKey primitives.Ed25519PublicKey
}

type NodeConfigValue struct {
	Uint32Value   uint32
	DurationValue time.Duration
}

type config struct {
	kv                      map[string]NodeConfigValue
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
	benchmarkConsensusRoundRetryIntervalMillis time.Duration,
	minimumTransactionsInBlock uint32,
	belowMinimalBlockDelayMillis time.Duration,
	queryGraceTimeoutMillis uint64,
) NodeConfig {
	cfg := &config{
		federationNodes:         federationNodes,
		nodePublicKey:           nodePublicKey,
		nodePrivateKey:          nodePrivateKey,
		constantConsensusLeader: constantConsensusLeader,
		activeConsensusAlgo:     activeConsensusAlgo,
		kv:                      make(map[string]NodeConfigValue),
	}

	cfg.SetUint32(VIRTUAL_CHAIN_ID, 42)
	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL_MILLIS, benchmarkConsensusRoundRetryIntervalMillis)

	cfg.SetDuration(QUERY_GRACE_TIMEOUT_MILLIS, time.Duration(queryGraceTimeoutMillis)*time.Millisecond)
	cfg.SetUint32(QUERY_SYNC_GRACE_BLOCK_DIST, 3)

	cfg.SetDuration(BLOCK_SYNC_COMMIT_TIMEOUT_MILLIS, 70*time.Millisecond)
	cfg.SetDuration(BLOCK_TRANSACTION_RECEIPT_QUERY_START_GRACE_SEC, 5*time.Second)
	cfg.SetDuration(BLOCK_TRANSACTION_RECEIPT_QUERY_END_GRACE_SEC, 5*time.Second)
	cfg.SetDuration(BLOCK_TRANSACTION_RECEIPT_QUERY_TRANSACTION_EXPIRE_SEC, 180*time.Second)

	cfg.SetUint32(STATE_HISTORY_RETENTION_IN_BLOCK_HEIGHTS, 5)

	cfg.SetDuration(BELOW_MINIMAL_BLOCK_DELAY_MILLIS, belowMinimalBlockDelayMillis)
	cfg.SetUint32(MINIMUM_TRANSACTION_IN_BLOCK, minimumTransactionsInBlock)

	cfg.SetUint32(STATE_HISTORY_RETENTION_IN_BLOCK_HEIGHTS, 5)

	cfg.SetUint32(PENDING_POOL_SIZE_IN_BYTES, 20*1024*1024)
	cfg.SetUint32(TRANSACTION_EXPIRATION_WINDOW_IN_SECONDS, 1800)
	cfg.SetUint32(FUTURE_TIMESTAMP_GRACE_IN_SECONDS, 180)

	return cfg
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

func (c *config) BenchmarkConsensusRoundRetryIntervalMillis() time.Duration {
	return c.kv[BENCHMARK_CONSENSUS_RETRY_INTERVAL_MILLIS].DurationValue

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

func (c *config) BelowMinimalBlockDelayMillis() time.Duration {
	return c.kv[BELOW_MINIMAL_BLOCK_DELAY_MILLIS].DurationValue
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

func (c *config) Set(key string, value NodeConfigValue) NodeConfig {
	c.kv[key] = value
	return c
}

func (c *config) SetDuration(key string, value time.Duration) NodeConfig {
	c.kv[key] = NodeConfigValue{DurationValue: value}
	return c
}

func (c *config) SetUint32(key string, value uint32) NodeConfig {
	c.kv[key] = NodeConfigValue{Uint32Value: value}
	return c
}

func (c *config) SetNodePublicKey(key primitives.Ed25519PublicKey) NodeConfig {
	c.nodePublicKey = key
	return c
}
