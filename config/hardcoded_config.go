// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"time"
)

type hardCodedValidatorNode struct {
	nodeAddress primitives.NodeAddress
}

type hardCodedGossipPeer struct {
	gossipPort     int
	gossipEndpoint string
	hexOrbsAddress string
}

type NodeConfigValue struct {
	Uint32Value   uint32
	DurationValue time.Duration
	StringValue   string
	BoolValue     bool
}

type MapBasedConfig struct {
	kv                      map[string]NodeConfigValue
	genesisValidatorNodes   map[string]ValidatorNode
	gossipPeers             map[string]GossipPeer
	nodeAddress             primitives.NodeAddress
	nodePrivateKey          primitives.EcdsaSecp256K1PrivateKey
	constantConsensusLeader primitives.NodeAddress
	activeConsensusAlgo     consensus.ConsensusAlgoType
}

const (
	PROTOCOL_VERSION = "PROTOCOL_VERSION"
	VIRTUAL_CHAIN_ID = "VIRTUAL_CHAIN_ID"
	NETWORK_TYPE     = "NETWORK_TYPE"

	BENCHMARK_CONSENSUS_RETRY_INTERVAL             = "BENCHMARK_CONSENSUS_RETRY_INTERVAL"
	BENCHMARK_CONSENSUS_REQUIRED_QUORUM_PERCENTAGE = "BENCHMARK_CONSENSUS_REQUIRED_QUORUM_PERCENTAGE"

	LEAN_HELIX_CONSENSUS_ROUND_TIMEOUT_INTERVAL = "LEAN_HELIX_CONSENSUS_ROUND_TIMEOUT_INTERVAL"
	LEAN_HELIX_CONSENSUS_MINIMUM_COMMITTEE_SIZE = "LEAN_HELIX_CONSENSUS_MINIMUM_COMMITTEE_SIZE"
	LEAN_HELIX_CONSENSUS_MAXIMUM_COMMITTEE_SIZE = "LEAN_HELIX_CONSENSUS_MAXIMUM_COMMITTEE_SIZE"
	INTER_NODE_SYNC_AUDIT_BLOCKS_YOUNGER_THAN   = "INTER_NODE_SYNC_AUDIT_BLOCKS_YOUNGER_THAN"
	LEAN_HELIX_SHOW_DEBUG                       = "LEAN_HELIX_SHOW_DEBUG"

	BLOCK_SYNC_NUM_BLOCKS_IN_BATCH      = "BLOCK_SYNC_NUM_BLOCKS_IN_BATCH"
	BLOCK_SYNC_NO_COMMIT_INTERVAL       = "BLOCK_SYNC_NO_COMMIT_INTERVAL"
	BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT = "BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT"
	BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT   = "BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT"

	BLOCK_STORAGE_TRANSACTION_RECEIPT_QUERY_TIMESTAMP_GRACE = "BLOCK_STORAGE_TRANSACTION_RECEIPT_QUERY_TIMESTAMP_GRACE"

	CONSENSUS_CONTEXT_MAXIMUM_TRANSACTIONS_IN_BLOCK   = "CONSENSUS_CONTEXT_MAXIMUM_TRANSACTIONS_IN_BLOCK"
	CONSENSUS_CONTEXT_SYSTEM_TIMESTAMP_ALLOWED_JITTER = "CONSENSUS_CONTEXT_SYSTEM_TIMESTAMP_ALLOWED_JITTER"

	STATE_STORAGE_HISTORY_SNAPSHOT_NUM = "STATE_STORAGE_HISTORY_SNAPSHOT_NUM"

	BLOCK_TRACKER_GRACE_DISTANCE = "BLOCK_TRACKER_GRACE_DISTANCE"
	BLOCK_TRACKER_GRACE_TIMEOUT  = "BLOCK_TRACKER_GRACE_TIMEOUT"

	TRANSACTION_POOL_PENDING_POOL_SIZE_IN_BYTES            = "TRANSACTION_POOL_PENDING_POOL_SIZE_IN_BYTES"
	TRANSACTION_EXPIRATION_WINDOW                          = "TRANSACTION_EXPIRATION_WINDOW"
	TRANSACTION_POOL_FUTURE_TIMESTAMP_GRACE_TIMEOUT        = "TRANSACTION_POOL_FUTURE_TIMESTAMP_GRACE_TIMEOUT"
	TRANSACTION_POOL_PENDING_POOL_CLEAR_EXPIRED_INTERVAL   = "TRANSACTION_POOL_PENDING_POOL_CLEAR_EXPIRED_INTERVAL"
	TRANSACTION_POOL_COMMITTED_POOL_CLEAR_EXPIRED_INTERVAL = "TRANSACTION_POOL_COMMITTED_POOL_CLEAR_EXPIRED_INTERVAL"
	TRANSACTION_POOL_PROPAGATION_BATCH_SIZE                = "TRANSACTION_POOL_PROPAGATION_BATCH_SIZE"
	TRANSACTION_POOL_PROPAGATION_BATCHING_TIMEOUT          = "TRANSACTION_POOL_PROPAGATION_BATCHING_TIMEOUT"
	TRANSACTION_POOL_TIME_BETWEEN_EMPTY_BLOCKS             = "TRANSACTION_POOL_TIME_BETWEEN_EMPTY_BLOCKS"
	TRANSACTION_POOL_NODE_SYNC_REJECT_TIME                 = "TRANSACTION_POOL_NODE_SYNC_REJECT_TIME"

	GOSSIP_LISTEN_PORT                    = "GOSSIP_LISTEN_PORT"
	GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL = "GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL"
	GOSSIP_NETWORK_TIMEOUT                = "GOSSIP_NETWORK_TIMEOUT"
	GOSSIP_RECONNECT_INTERVAL             = "GOSSIP_RECONNECT_INTERVAL"

	PUBLIC_API_SEND_TRANSACTION_TIMEOUT = "PUBLIC_API_SEND_TRANSACTION_TIMEOUT"
	PUBLIC_API_NODE_SYNC_WARNING_TIME   = "PUBLIC_API_NODE_SYNC_WARNING_TIME"

	PROCESSOR_ARTIFACT_PATH               = "PROCESSOR_ARTIFACT_PATH"
	PROCESSOR_SANITIZE_DEPLOYED_CONTRACTS = "PROCESSOR_SANITIZE_DEPLOYED_CONTRACTS"
	PROCESSOR_PERFORM_WARM_UP_COMPILATION = "PROCESSOR_PERFORM_WARM_UP_COMPILATION"

	METRICS_REPORT_INTERVAL = "METRICS_REPORT_INTERVAL"

	ETHEREUM_ENDPOINT                  = "ETHEREUM_ENDPOINT"
	ETHEREUM_FINALITY_TIME_COMPONENT   = "ETHEREUM_FINALITY_TIME_COMPONENT"
	ETHEREUM_FINALITY_BLOCKS_COMPONENT = "ETHEREUM_FINALITY_BLOCKS_COMPONENT"

	LOGGER_HTTP_ENDPOINT            = "LOGGER_HTTP_ENDPOINT"
	LOGGER_BULK_SIZE                = "LOGGER_BULK_SIZE"
	LOGGER_FILE_TRUNCATION_INTERVAL = "LOGGER_FILE_TRUNCATION_INTERVAL"
	LOGGER_FULL_LOG                 = "LOGGER_FULL_LOG"

	BLOCK_STORAGE_FILE_SYSTEM_DATA_DIR                = "BLOCK_STORAGE_FILE_SYSTEM_DATA_DIR"
	BLOCK_STORAGE_FILE_SYSTEM_MAX_BLOCK_SIZE_IN_BYTES = "BLOCK_STORAGE_FILE_SYSTEM_MAX_BLOCK_SIZE_IN_BYTES"

	PROFILING = "PROFILING"

	HTTP_ADDRESS = "HTTP_ADDRESS"

	NTP_ENDPOINT = "NTP_ENDPOINT"

	SIGNER_ENDPOINT = "SIGNER_ENDPOINT"
)

func NewHardCodedValidatorNode(nodeAddress primitives.NodeAddress) ValidatorNode {
	return &hardCodedValidatorNode{
		nodeAddress: nodeAddress,
	}
}

func NewHardCodedGossipPeer(gossipPort int, gossipEndpoint string, hexAddress string) GossipPeer {
	return &hardCodedGossipPeer{
		gossipPort:     gossipPort,
		gossipEndpoint: gossipEndpoint,
		hexOrbsAddress: hexAddress,
	}
}

func (c *MapBasedConfig) Set(key string, value NodeConfigValue) *MapBasedConfig {
	c.kv[key] = value
	return c
}

func (c *MapBasedConfig) SetDuration(key string, value time.Duration) *MapBasedConfig {
	c.kv[key] = NodeConfigValue{DurationValue: value}
	return c
}

func (c *MapBasedConfig) SetUint32(key string, value uint32) *MapBasedConfig {
	c.kv[key] = NodeConfigValue{Uint32Value: value}
	return c
}

func (c *MapBasedConfig) SetString(key string, value string) *MapBasedConfig {
	c.kv[key] = NodeConfigValue{StringValue: value}
	return c
}

func (c *MapBasedConfig) SetBool(key string, value bool) *MapBasedConfig {
	c.kv[key] = NodeConfigValue{BoolValue: value}
	return c
}

func (c *MapBasedConfig) SetNodeAddress(key primitives.NodeAddress) *MapBasedConfig {
	c.nodeAddress = key
	return c
}

func (c *MapBasedConfig) SetNodePrivateKey(key primitives.EcdsaSecp256K1PrivateKey) *MapBasedConfig {
	c.nodePrivateKey = key
	return c
}

func (c *MapBasedConfig) SetBenchmarkConsensusConstantLeader(key primitives.NodeAddress) *MapBasedConfig {
	c.constantConsensusLeader = key
	return c
}

func (c *MapBasedConfig) SetActiveConsensusAlgo(algoType consensus.ConsensusAlgoType) *MapBasedConfig {
	c.activeConsensusAlgo = algoType
	return c
}

func (c *MapBasedConfig) SetGenesisValidatorNodes(nodes map[string]ValidatorNode) *MapBasedConfig {
	c.genesisValidatorNodes = nodes
	return c
}

func (c *MapBasedConfig) SetGossipPeers(gossipPeers map[string]GossipPeer) *MapBasedConfig {
	c.gossipPeers = gossipPeers
	return c
}

func (c *hardCodedValidatorNode) NodeAddress() primitives.NodeAddress {
	return c.nodeAddress
}

func (c *hardCodedGossipPeer) GossipPort() int {
	return c.gossipPort
}

func (c *hardCodedGossipPeer) GossipEndpoint() string {
	return c.gossipEndpoint
}

func (c *hardCodedGossipPeer) HexOrbsAddress() string {
	return c.hexOrbsAddress
}

func (c *MapBasedConfig) NodeAddress() primitives.NodeAddress {
	return c.nodeAddress
}

func (c *MapBasedConfig) NodePrivateKey() primitives.EcdsaSecp256K1PrivateKey {
	return c.nodePrivateKey
}

func (c *MapBasedConfig) ProtocolVersion() primitives.ProtocolVersion {
	return primitives.ProtocolVersion(c.kv[PROTOCOL_VERSION].Uint32Value)
}

func (c *MapBasedConfig) VirtualChainId() primitives.VirtualChainId {
	return primitives.VirtualChainId(c.kv[VIRTUAL_CHAIN_ID].Uint32Value)
}

func (c *MapBasedConfig) NetworkType() protocol.SignerNetworkType {
	return protocol.SignerNetworkType(c.kv[NETWORK_TYPE].Uint32Value)
}

func (c *MapBasedConfig) GenesisValidatorNodes() map[string]ValidatorNode {
	return c.genesisValidatorNodes
}

func (c *MapBasedConfig) GossipPeers() map[string]GossipPeer {
	return c.gossipPeers
}

func (c *MapBasedConfig) BenchmarkConsensusConstantLeader() primitives.NodeAddress {
	return c.constantConsensusLeader
}

func (c *MapBasedConfig) ActiveConsensusAlgo() consensus.ConsensusAlgoType {
	return c.activeConsensusAlgo
}

func (c *MapBasedConfig) BenchmarkConsensusRetryInterval() time.Duration {
	return c.kv[BENCHMARK_CONSENSUS_RETRY_INTERVAL].DurationValue
}

func (c *MapBasedConfig) LeanHelixConsensusRoundTimeoutInterval() time.Duration {
	return c.kv[LEAN_HELIX_CONSENSUS_ROUND_TIMEOUT_INTERVAL].DurationValue
}

func (c *MapBasedConfig) LeanHelixShowDebug() bool {
	return c.kv[LEAN_HELIX_SHOW_DEBUG].BoolValue
}

func (c *MapBasedConfig) BlockSyncNumBlocksInBatch() uint32 {
	return c.kv[BLOCK_SYNC_NUM_BLOCKS_IN_BATCH].Uint32Value
}

func (c *MapBasedConfig) BlockSyncNoCommitInterval() time.Duration {
	return c.kv[BLOCK_SYNC_NO_COMMIT_INTERVAL].DurationValue
}

func (c *MapBasedConfig) BlockSyncCollectResponseTimeout() time.Duration {
	return c.kv[BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT].DurationValue
}

func (c *MapBasedConfig) BlockStorageTransactionReceiptQueryTimestampGrace() time.Duration {
	return c.kv[BLOCK_STORAGE_TRANSACTION_RECEIPT_QUERY_TIMESTAMP_GRACE].DurationValue
}

func (c *MapBasedConfig) ConsensusContextMaximumTransactionsInBlock() uint32 {
	return c.kv[CONSENSUS_CONTEXT_MAXIMUM_TRANSACTIONS_IN_BLOCK].Uint32Value
}

func (c *MapBasedConfig) ConsensusContextSystemTimestampAllowedJitter() time.Duration {
	return c.kv[CONSENSUS_CONTEXT_SYSTEM_TIMESTAMP_ALLOWED_JITTER].DurationValue
}

func (c *MapBasedConfig) StateStorageHistorySnapshotNum() uint32 {
	return c.kv[STATE_STORAGE_HISTORY_SNAPSHOT_NUM].Uint32Value
}

func (c *MapBasedConfig) BlockTrackerGraceDistance() uint32 {
	return c.kv[BLOCK_TRACKER_GRACE_DISTANCE].Uint32Value
}

func (c *MapBasedConfig) BlockTrackerGraceTimeout() time.Duration {
	return c.kv[BLOCK_TRACKER_GRACE_TIMEOUT].DurationValue
}

func (c *MapBasedConfig) TransactionPoolPendingPoolSizeInBytes() uint32 {
	return c.kv[TRANSACTION_POOL_PENDING_POOL_SIZE_IN_BYTES].Uint32Value
}

func (c *MapBasedConfig) TransactionExpirationWindow() time.Duration {
	return c.kv[TRANSACTION_EXPIRATION_WINDOW].DurationValue
}

func (c *MapBasedConfig) TransactionPoolFutureTimestampGraceTimeout() time.Duration {
	return c.kv[TRANSACTION_POOL_FUTURE_TIMESTAMP_GRACE_TIMEOUT].DurationValue
}

func (c *MapBasedConfig) TransactionPoolPendingPoolClearExpiredInterval() time.Duration {
	return c.kv[TRANSACTION_POOL_PENDING_POOL_CLEAR_EXPIRED_INTERVAL].DurationValue
}

func (c *MapBasedConfig) TransactionPoolCommittedPoolClearExpiredInterval() time.Duration {
	return c.kv[TRANSACTION_POOL_COMMITTED_POOL_CLEAR_EXPIRED_INTERVAL].DurationValue
}

func (c *MapBasedConfig) TransactionPoolPropagationBatchSize() uint16 {
	return uint16(c.kv[TRANSACTION_POOL_PROPAGATION_BATCH_SIZE].Uint32Value)
}

func (c *MapBasedConfig) TransactionPoolPropagationBatchingTimeout() time.Duration {
	return c.kv[TRANSACTION_POOL_PROPAGATION_BATCHING_TIMEOUT].DurationValue
}

func (c *MapBasedConfig) TransactionPoolTimeBetweenEmptyBlocks() time.Duration {
	return c.kv[TRANSACTION_POOL_TIME_BETWEEN_EMPTY_BLOCKS].DurationValue
}

func (c *MapBasedConfig) TransactionPoolNodeSyncRejectTime() time.Duration {
	return c.kv[TRANSACTION_POOL_NODE_SYNC_REJECT_TIME].DurationValue
}

func (c *MapBasedConfig) PublicApiSendTransactionTimeout() time.Duration {
	return c.kv[PUBLIC_API_SEND_TRANSACTION_TIMEOUT].DurationValue
}

func (c *MapBasedConfig) PublicApiNodeSyncWarningTime() time.Duration {
	return c.kv[PUBLIC_API_NODE_SYNC_WARNING_TIME].DurationValue
}

func (c *MapBasedConfig) BlockSyncCollectChunksTimeout() time.Duration {
	return c.kv[BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT].DurationValue
}

func (c *MapBasedConfig) ProcessorArtifactPath() string {
	return c.kv[PROCESSOR_ARTIFACT_PATH].StringValue
}

func (c *MapBasedConfig) ProcessorSanitizeDeployedContracts() bool {
	return c.kv[PROCESSOR_SANITIZE_DEPLOYED_CONTRACTS].BoolValue
}

func (c *MapBasedConfig) ProcessorPerformWarmUpCompilation() bool {
	return c.kv[PROCESSOR_PERFORM_WARM_UP_COMPILATION].BoolValue
}

func (c *MapBasedConfig) GossipListenPort() uint16 {
	return uint16(c.kv[GOSSIP_LISTEN_PORT].Uint32Value)
}

func (c *MapBasedConfig) GossipConnectionKeepAliveInterval() time.Duration {
	return c.kv[GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL].DurationValue
}

func (c *MapBasedConfig) GossipNetworkTimeout() time.Duration {
	return c.kv[GOSSIP_NETWORK_TIMEOUT].DurationValue
}

func (c *MapBasedConfig) GossipReconnectInterval() time.Duration {
	return c.kv[GOSSIP_RECONNECT_INTERVAL].DurationValue
}

func (c *MapBasedConfig) BenchmarkConsensusRequiredQuorumPercentage() uint32 {
	return c.kv[BENCHMARK_CONSENSUS_REQUIRED_QUORUM_PERCENTAGE].Uint32Value
}

func (c *MapBasedConfig) LeanHelixConsensusMinimumCommitteeSize() uint32 {
	return c.kv[LEAN_HELIX_CONSENSUS_MINIMUM_COMMITTEE_SIZE].Uint32Value
}

func (c *MapBasedConfig) LeanHelixConsensusMaximumCommitteeSize() uint32 {
	return c.kv[LEAN_HELIX_CONSENSUS_MAXIMUM_COMMITTEE_SIZE].Uint32Value
}

func (c *MapBasedConfig) InterNodeSyncAuditBlocksYoungerThan() time.Duration {
	return c.kv[INTER_NODE_SYNC_AUDIT_BLOCKS_YOUNGER_THAN].DurationValue
}

func (c *MapBasedConfig) EthereumEndpoint() string {
	return c.kv[ETHEREUM_ENDPOINT].StringValue
}

func (c *MapBasedConfig) EthereumFinalityTimeComponent() time.Duration {
	return c.kv[ETHEREUM_FINALITY_TIME_COMPONENT].DurationValue
}

func (c *MapBasedConfig) EthereumFinalityBlocksComponent() uint32 {
	return c.kv[ETHEREUM_FINALITY_BLOCKS_COMPONENT].Uint32Value
}

func (c *MapBasedConfig) LoggerHttpEndpoint() string {
	return c.kv[LOGGER_HTTP_ENDPOINT].StringValue
}

func (c *MapBasedConfig) LoggerBulkSize() uint32 {
	return c.kv[LOGGER_BULK_SIZE].Uint32Value
}

func (c *MapBasedConfig) LoggerFileTruncationInterval() time.Duration {
	return c.kv[LOGGER_FILE_TRUNCATION_INTERVAL].DurationValue
}

func (c *MapBasedConfig) LoggerFullLog() bool {
	return c.kv[LOGGER_FULL_LOG].BoolValue
}

func (c *MapBasedConfig) BlockStorageFileSystemDataDir() string {
	return c.kv[BLOCK_STORAGE_FILE_SYSTEM_DATA_DIR].StringValue
}

func (c *MapBasedConfig) BlockStorageFileSystemMaxBlockSizeInBytes() uint32 {
	return c.kv[BLOCK_STORAGE_FILE_SYSTEM_MAX_BLOCK_SIZE_IN_BYTES].Uint32Value
}

func (c *MapBasedConfig) Profiling() bool {
	return c.kv[PROFILING].BoolValue
}

func (c *MapBasedConfig) HttpAddress() string {
	return c.kv[HTTP_ADDRESS].StringValue
}

func (c *MapBasedConfig) NTPEndpoint() string {
	return c.kv[NTP_ENDPOINT].StringValue
}

func (c *MapBasedConfig) SignerEndpoint() string {
	return c.kv[SIGNER_ENDPOINT].StringValue
}
