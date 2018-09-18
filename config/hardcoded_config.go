package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"time"
)

type hardCodedFederationNode struct {
	nodePublicKey primitives.Ed25519PublicKey
}

type hardCodedGossipPeer struct {
	gossipPort     uint16
	gossipEndpoint string
}

type NodeConfigValue struct {
	Uint32Value   uint32
	Uint16Value   uint16
	DurationValue time.Duration
}

type config struct {
	kv                      map[string]NodeConfigValue
	federationNodes         map[string]FederationNode
	gossipPeers             map[string]GossipPeer
	nodePublicKey           primitives.Ed25519PublicKey
	nodePrivateKey          primitives.Ed25519PrivateKey
	constantConsensusLeader primitives.Ed25519PublicKey
	activeConsensusAlgo     consensus.ConsensusAlgoType
}

const (
	VIRTUAL_CHAIN_ID                   = "VIRTUAL_CHAIN_ID"
	BENCHMARK_CONSENSUS_RETRY_INTERVAL = "BENCHMARK_CONSENSUS_RETRY_INTERVAL"

	BLOCK_SYNC_BATCH_SIZE               = "BLOCK_SYNC_BATCH_SIZE"
	BLOCK_SYNC_INTERVAL                 = "BLOCK_SYNC_INTERVAL"
	BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT = "BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT"

	BLOCK_TRANSACTION_RECEIPT_QUERY_GRACE_START       = "BLOCK_TRANSACTION_RECEIPT_QUERY_GRACE_START"
	BLOCK_TRANSACTION_RECEIPT_QUERY_GRACE_END         = "BLOCK_TRANSACTION_RECEIPT_QUERY_GRACE_END"
	BLOCK_TRANSACTION_RECEIPT_QUERY_EXPIRATION_WINDOW = "BLOCK_TRANSACTION_RECEIPT_QUERY_EXPIRATION_WINDOW"
	BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT                 = "BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT"

	CONSENSUS_CONTEXT_MINIMAL_BLOCK_DELAY          = "CONSENSUS_CONTEXT_MINIMAL_BLOCK_DELAY"
	CONSENSUS_CONTEXT_MINIMUM_TRANSACTION_IN_BLOCK = "CONSENSUS_CONTEXT_MINIMUM_TRANSACTION_IN_BLOCK"

	STATE_STORAGE_HISTORY_RETENTION_DISTANCE = "STATE_STORAGE_HISTORY_RETENTION_DISTANCE"

	BLOCK_TRACKER_GRACE_DISTANCE = "BLOCK_TRACKER_GRACE_DISTANCE"
	BLOCK_TRACKER_GRACE_TIMEOUT  = "BLOCK_TRACKER_GRACE_TIMEOUT"

	TRANSACTION_POOL_PENDING_POOL_SIZE_IN_BYTES            = "TRANSACTION_POOL_PENDING_POOL_SIZE_IN_BYTES"
	TRANSACTION_POOL_TRANSACTION_EXPIRATION_WINDOW         = "TRANSACTION_POOL_TRANSACTION_EXPIRATION_WINDOW"
	TRANSACTION_POOL_FUTURE_TIMESTAMP_GRACE_TIMEOUT        = "TRANSACTION_POOL_FUTURE_TIMESTAMP_GRACE_TIMEOUT"
	TRANSACTION_POOL_PENDING_POOL_CLEAR_EXPIRED_INTERVAL   = "TRANSACTION_POOL_PENDING_POOL_CLEAR_EXPIRED_INTERVAL"
	TRANSACTION_POOL_COMMITTED_POOL_CLEAR_EXPIRED_INTERVAL = "TRANSACTION_POOL_COMMITTED_POOL_CLEAR_EXPIRED_INTERVAL"

	GOSSIP_LISTEN_PORT                    = "GOSSIP_LISTEN_PORT"
	GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL = "GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL"
	GOSSIP_NETWORK_TIMEOUT                = "GOSSIP_NETWORK_TIMEOUT"

	PUBLIC_API_SEND_TRANSACTION_TIMEOUT = "PUBLIC_API_SEND_TRANSACTION_TIMEOUT"
	PUBLIC_API_TRANSACTION_STATUS_GRACE = "PUBLIC_API_TRANSACTION_STATUS_GRACE"
)

func NewHardCodedFederationNode(nodePublicKey primitives.Ed25519PublicKey) FederationNode {
	return &hardCodedFederationNode{
		nodePublicKey: nodePublicKey,
	}
}

func NewHardCodedGossipPeer(gossipPort uint16, gossipEndpoint string) GossipPeer {
	return &hardCodedGossipPeer{
		gossipPort:     gossipPort,
		gossipEndpoint: gossipEndpoint,
	}
}

func newHardCodedConfig(
	federationNodes map[string]FederationNode,
	gossipPeers map[string]GossipPeer,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
) NodeConfig {
	cfg := &config{
		federationNodes:         federationNodes,
		gossipPeers:             gossipPeers,
		nodePublicKey:           nodePublicKey,
		nodePrivateKey:          nodePrivateKey,
		constantConsensusLeader: constantConsensusLeader,
		activeConsensusAlgo:     activeConsensusAlgo,
		kv:                      make(map[string]NodeConfigValue),
	}

	cfg.SetUint32(VIRTUAL_CHAIN_ID, 42)

	cfg.SetUint32(BLOCK_TRACKER_GRACE_DISTANCE, 3)

	cfg.SetDuration(BLOCK_SYNC_BATCH_SIZE, 10000)
	cfg.SetDuration(BLOCK_SYNC_INTERVAL, 5*time.Second)
	cfg.SetDuration(BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT, 3*time.Second)
	cfg.SetDuration(BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT, 5*time.Second)

	cfg.SetDuration(BLOCK_TRANSACTION_RECEIPT_QUERY_GRACE_START, 5*time.Second)
	cfg.SetDuration(BLOCK_TRANSACTION_RECEIPT_QUERY_GRACE_END, 5*time.Second)
	cfg.SetDuration(BLOCK_TRANSACTION_RECEIPT_QUERY_EXPIRATION_WINDOW, 3*time.Minute)

	cfg.SetUint32(STATE_STORAGE_HISTORY_RETENTION_DISTANCE, 5)

	cfg.SetUint32(CONSENSUS_CONTEXT_MINIMUM_TRANSACTION_IN_BLOCK, uint32(1))

	cfg.SetUint32(TRANSACTION_POOL_PENDING_POOL_SIZE_IN_BYTES, 20*1024*1024)
	cfg.SetDuration(TRANSACTION_POOL_TRANSACTION_EXPIRATION_WINDOW, 30*time.Minute)
	cfg.SetDuration(TRANSACTION_POOL_PENDING_POOL_CLEAR_EXPIRED_INTERVAL, 10*time.Second)
	cfg.SetDuration(TRANSACTION_POOL_COMMITTED_POOL_CLEAR_EXPIRED_INTERVAL, 30*time.Second)

	return cfg
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

func (c *config) SetUint16(key string, value uint16) NodeConfig {
	c.kv[key] = NodeConfigValue{Uint16Value: value}
	return c
}

func (c *config) SetNodePublicKey(key primitives.Ed25519PublicKey) NodeConfig {
	c.nodePublicKey = key
	return c
}

func (c *config) SetNodePrivateKey(key primitives.Ed25519PrivateKey) NodeConfig {
	c.nodePrivateKey = key
	return c
}

func (c *config) SetFederationNodes(federationNodes map[string]FederationNode) NodeConfig {
	c.federationNodes = federationNodes
	return c
}

func (c *config) SetGossipPeers(gossipPeers map[string]GossipPeer) NodeConfig {
	c.gossipPeers = gossipPeers
	return c
}

func (c *hardCodedFederationNode) NodePublicKey() primitives.Ed25519PublicKey {
	return c.nodePublicKey
}

func (c *hardCodedGossipPeer) GossipPort() uint16 {
	return c.gossipPort
}

func (c *hardCodedGossipPeer) GossipEndpoint() string {
	return c.gossipEndpoint
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

func (c *config) GossipPeers(asOfBlock uint64) map[string]GossipPeer {
	return c.gossipPeers
}

func (c *config) ConstantConsensusLeader() primitives.Ed25519PublicKey {
	return c.constantConsensusLeader
}

func (c *config) ActiveConsensusAlgo() consensus.ConsensusAlgoType {
	return c.activeConsensusAlgo
}

func (c *config) BenchmarkConsensusRetryInterval() time.Duration {
	return c.kv[BENCHMARK_CONSENSUS_RETRY_INTERVAL].DurationValue
}

func (c *config) BlockSyncBatchSize() uint32 {
	return c.kv[BLOCK_SYNC_BATCH_SIZE].Uint32Value
}

func (c *config) BlockSyncInterval() time.Duration {
	return c.kv[BLOCK_SYNC_INTERVAL].DurationValue
}

func (c *config) BlockSyncCollectResponseTimeout() time.Duration {
	return c.kv[BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT].DurationValue
}

func (c *config) BlockTransactionReceiptQueryGraceStart() time.Duration {
	return c.kv[BLOCK_TRANSACTION_RECEIPT_QUERY_GRACE_START].DurationValue
}

func (c *config) BlockTransactionReceiptQueryGraceEnd() time.Duration {
	return c.kv[BLOCK_TRANSACTION_RECEIPT_QUERY_GRACE_END].DurationValue
}

func (c *config) BlockTransactionReceiptQueryExpirationWindow() time.Duration {
	return c.kv[BLOCK_TRANSACTION_RECEIPT_QUERY_EXPIRATION_WINDOW].DurationValue
}

func (c *config) ConsensusContextMinimalBlockDelay() time.Duration {
	return c.kv[CONSENSUS_CONTEXT_MINIMAL_BLOCK_DELAY].DurationValue
}

func (c *config) ConsensusContextMinimumTransactionsInBlock() uint32 {
	return c.kv[CONSENSUS_CONTEXT_MINIMUM_TRANSACTION_IN_BLOCK].Uint32Value
}

func (c *config) StateStorageHistoryRetentionDistance() uint32 {
	return c.kv[STATE_STORAGE_HISTORY_RETENTION_DISTANCE].Uint32Value
}

func (c *config) BlockTrackerGraceDistance() uint32 {
	return c.kv[BLOCK_TRACKER_GRACE_DISTANCE].Uint32Value
}

func (c *config) BlockTrackerGraceTimeout() time.Duration {
	return c.kv[BLOCK_TRACKER_GRACE_TIMEOUT].DurationValue
}

func (c *config) TransactionPoolPendingPoolSizeInBytes() uint32 {
	return c.kv[TRANSACTION_POOL_PENDING_POOL_SIZE_IN_BYTES].Uint32Value
}

func (c *config) TransactionPoolTransactionExpirationWindow() time.Duration {
	return c.kv[TRANSACTION_POOL_TRANSACTION_EXPIRATION_WINDOW].DurationValue
}

func (c *config) TransactionPoolFutureTimestampGraceTimeout() time.Duration {
	return c.kv[TRANSACTION_POOL_FUTURE_TIMESTAMP_GRACE_TIMEOUT].DurationValue
}

func (c *config) TransactionPoolPendingPoolClearExpiredInterval() time.Duration {
	return c.kv[TRANSACTION_POOL_PENDING_POOL_CLEAR_EXPIRED_INTERVAL].DurationValue
}

func (c *config) TransactionPoolCommittedPoolClearExpiredInterval() time.Duration {
	return c.kv[TRANSACTION_POOL_COMMITTED_POOL_CLEAR_EXPIRED_INTERVAL].DurationValue
}

func (c *config) SendTransactionTimeout() time.Duration {
	return c.kv[PUBLIC_API_SEND_TRANSACTION_TIMEOUT].DurationValue
}

func (c *config) GetTransactionStatusGrace() time.Duration {
	return c.kv[PUBLIC_API_TRANSACTION_STATUS_GRACE].DurationValue
}

func (c *config) BlockSyncCollectChunksTimeout() time.Duration {
	return c.kv[BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT].DurationValue
}

func (c *config) GossipListenPort() uint16 {
	return c.kv[GOSSIP_LISTEN_PORT].Uint16Value
}

func (c *config) GossipConnectionKeepAliveInterval() time.Duration {
	return c.kv[GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL].DurationValue
}

func (c *config) GossipNetworkTimeout() time.Duration {
	return c.kv[GOSSIP_NETWORK_TIMEOUT].DurationValue
}
