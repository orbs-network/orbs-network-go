package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"time"
)

type NodeConfig interface {
	// shared
	ProtocolVersion() primitives.ProtocolVersion
	VirtualChainId() primitives.VirtualChainId
	NodeAddress() primitives.NodeAddress
	NodePrivateKey() primitives.EcdsaSecp256K1PrivateKey
	NetworkSize(asOfBlock uint64) uint32
	FederationNodes(asOfBlock uint64) map[string]FederationNode
	GossipPeers(asOfBlock uint64) map[string]GossipPeer

	// consensus
	ConstantConsensusLeader() primitives.NodeAddress
	ActiveConsensusAlgo() consensus.ConsensusAlgoType
	ConsensusRequiredQuorumPercentage() uint32
	ConsensusMinimumCommitteeSize() uint32

	// Lean Helix consensus
	LeanHelixConsensusRoundTimeoutInterval() time.Duration

	// benchmark consensus
	BenchmarkConsensusRetryInterval() time.Duration

	// block storage
	BlockSyncBatchSize() uint32
	BlockSyncNoCommitInterval() time.Duration
	BlockSyncCollectResponseTimeout() time.Duration
	BlockTransactionReceiptQueryGraceStart() time.Duration
	BlockTransactionReceiptQueryGraceEnd() time.Duration
	BlockTransactionReceiptQueryExpirationWindow() time.Duration
	BlockSyncCollectChunksTimeout() time.Duration

	// file system block storage
	BlockStorageDataDir() string

	// state storage
	StateStorageHistorySnapshotNum() uint32

	// block tracker
	BlockTrackerGraceDistance() uint32
	BlockTrackerGraceTimeout() time.Duration

	// consensus context
	ConsensusContextMaximumTransactionsInBlock() uint32
	ConsensusContextSystemTimestampAllowedJitter() time.Duration

	// transaction pool
	TransactionPoolPendingPoolSizeInBytes() uint32
	TransactionPoolTransactionExpirationWindow() time.Duration
	TransactionPoolFutureTimestampGraceTimeout() time.Duration
	TransactionPoolPendingPoolClearExpiredInterval() time.Duration
	TransactionPoolCommittedPoolClearExpiredInterval() time.Duration
	TransactionPoolPropagationBatchSize() uint16
	TransactionPoolPropagationBatchingTimeout() time.Duration

	// gossip
	GossipListenPort() uint16
	GossipConnectionKeepAliveInterval() time.Duration
	GossipNetworkTimeout() time.Duration

	// public api
	SendTransactionTimeout() time.Duration

	// processor
	ProcessorArtifactPath() string

	// metrics
	MetricsReportInterval() time.Duration

	// ethereum connector (crosschain)
	EthereumEndpoint() string

	// Logger
	LoggerHttpEndpoint() string
	LoggerBulkSize() uint32
}

type OverridableConfig interface {
	NodeConfig
	OverrideNodeSpecificValues(gossipListenPort int, nodeAddress primitives.NodeAddress, nodePrivateKey primitives.EcdsaSecp256K1PrivateKey, blockStorageDataDirPrefix string) NodeConfig
}

type mutableNodeConfig interface {
	OverridableConfig
	Set(key string, value NodeConfigValue) mutableNodeConfig
	SetDuration(key string, value time.Duration) mutableNodeConfig
	SetUint32(key string, value uint32) mutableNodeConfig
	SetString(key string, value string) mutableNodeConfig
	SetFederationNodes(nodes map[string]FederationNode) mutableNodeConfig
	SetGossipPeers(peers map[string]GossipPeer) mutableNodeConfig
	SetNodeAddress(key primitives.NodeAddress) mutableNodeConfig
	SetNodePrivateKey(key primitives.EcdsaSecp256K1PrivateKey) mutableNodeConfig
	SetConstantConsensusLeader(key primitives.NodeAddress) mutableNodeConfig
	SetActiveConsensusAlgo(algoType consensus.ConsensusAlgoType) mutableNodeConfig
	MergeWithFileConfig(source string) (mutableNodeConfig, error)
	Clone() mutableNodeConfig
}

type BlockStorageConfig interface {
	NodeAddress() primitives.NodeAddress
	BlockSyncBatchSize() uint32
	BlockSyncNoCommitInterval() time.Duration
	BlockSyncCollectResponseTimeout() time.Duration
	BlockSyncCollectChunksTimeout() time.Duration
	BlockTransactionReceiptQueryGraceStart() time.Duration
	BlockTransactionReceiptQueryGraceEnd() time.Duration
	BlockTransactionReceiptQueryExpirationWindow() time.Duration
}

type FilesystemBlockPersistenceConfig interface {
	BlockStorageDataDir() string
}

type GossipTransportConfig interface {
	NodeAddress() primitives.NodeAddress
	GossipPeers(asOfBlock uint64) map[string]GossipPeer
	GossipListenPort() uint16
	GossipConnectionKeepAliveInterval() time.Duration
	GossipNetworkTimeout() time.Duration
}

// Config based on https://github.com/orbs-network/orbs-spec/blob/master/behaviors/config/services.md#consensus-context
type ConsensusContextConfig interface {
	ProtocolVersion() primitives.ProtocolVersion
	VirtualChainId() primitives.VirtualChainId
	ConsensusContextMaximumTransactionsInBlock() uint32
	FederationNodes(asOfBlock uint64) map[string]FederationNode
	ConsensusMinimumCommitteeSize() uint32
	ConsensusContextSystemTimestampAllowedJitter() time.Duration
}

type PublicApiConfig interface {
	SendTransactionTimeout() time.Duration
	VirtualChainId() primitives.VirtualChainId
}

type StateStorageConfig interface {
	StateStorageHistorySnapshotNum() uint32
	BlockTrackerGraceDistance() uint32
	BlockTrackerGraceTimeout() time.Duration
}

type TransactionPoolConfig interface {
	NodeAddress() primitives.NodeAddress
	NodePrivateKey() primitives.EcdsaSecp256K1PrivateKey
	VirtualChainId() primitives.VirtualChainId
	BlockTrackerGraceDistance() uint32
	BlockTrackerGraceTimeout() time.Duration
	TransactionPoolPendingPoolSizeInBytes() uint32
	TransactionPoolTransactionExpirationWindow() time.Duration
	TransactionPoolFutureTimestampGraceTimeout() time.Duration
	TransactionPoolPendingPoolClearExpiredInterval() time.Duration
	TransactionPoolCommittedPoolClearExpiredInterval() time.Duration
	TransactionPoolPropagationBatchSize() uint16
	TransactionPoolPropagationBatchingTimeout() time.Duration
}

type FederationNode interface {
	NodeAddress() primitives.NodeAddress
}

type GossipPeer interface {
	GossipPort() int
	GossipEndpoint() string
}
