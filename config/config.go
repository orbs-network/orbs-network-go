package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"time"
)

type NodeConfig interface {
	// shared
	VirtualChainId() primitives.VirtualChainId
	NodePublicKey() primitives.Ed25519PublicKey
	NodePrivateKey() primitives.Ed25519PrivateKey
	NetworkSize(asOfBlock uint64) uint32
	FederationNodes(asOfBlock uint64) map[string]FederationNode
	GossipPeers(asOfBlock uint64) map[string]GossipPeer

	// consensus
	ConstantConsensusLeader() primitives.Ed25519PublicKey
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

	// state storage
	StateStorageHistorySnapshotNum() uint32

	// block tracker
	BlockTrackerGraceDistance() uint32
	BlockTrackerGraceTimeout() time.Duration

	// consensus context
	ConsensusContextMinimalBlockTime() time.Duration
	ConsensusContextMinimumTransactionsInBlock() uint32
	ConsensusContextMaximumTransactionsInBlock() uint32

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

	// ethereum connector (sidechain)
	EthereumEndpoint() string
}

type OverridableConfig interface {
	NodeConfig
	OverrideNodeSpecificValues(
		gossipListenPort int,
		nodePublicKey primitives.Ed25519PublicKey,
		nodePrivateKey primitives.Ed25519PrivateKey) NodeConfig
}

type mutableNodeConfig interface {
	OverridableConfig
	Set(key string, value NodeConfigValue) mutableNodeConfig
	SetDuration(key string, value time.Duration) mutableNodeConfig
	SetUint32(key string, value uint32) mutableNodeConfig
	SetString(key string, value string) mutableNodeConfig
	SetFederationNodes(nodes map[string]FederationNode) mutableNodeConfig
	SetGossipPeers(peers map[string]GossipPeer) mutableNodeConfig
	SetNodePublicKey(key primitives.Ed25519PublicKey) mutableNodeConfig
	SetNodePrivateKey(key primitives.Ed25519PrivateKey) mutableNodeConfig
	SetConstantConsensusLeader(key primitives.Ed25519PublicKey) mutableNodeConfig
	SetActiveConsensusAlgo(algoType consensus.ConsensusAlgoType) mutableNodeConfig
	MergeWithFileConfig(source string) (mutableNodeConfig, error)
	Clone() mutableNodeConfig
}

type BlockStorageConfig interface {
	NodePublicKey() primitives.Ed25519PublicKey
	BlockSyncBatchSize() uint32
	BlockSyncNoCommitInterval() time.Duration
	BlockSyncCollectResponseTimeout() time.Duration
	BlockSyncCollectChunksTimeout() time.Duration
	BlockTransactionReceiptQueryGraceStart() time.Duration
	BlockTransactionReceiptQueryGraceEnd() time.Duration
	BlockTransactionReceiptQueryExpirationWindow() time.Duration
}

type GossipTransportConfig interface {
	NodePublicKey() primitives.Ed25519PublicKey
	GossipPeers(asOfBlock uint64) map[string]GossipPeer
	GossipListenPort() uint16
	GossipConnectionKeepAliveInterval() time.Duration
	GossipNetworkTimeout() time.Duration
}

// TODO See if more config props needed here, based on:
// https://github.com/orbs-network/orbs-spec/blob/master/behaviors/config/services.md#consensus-context
type ConsensusContextConfig interface {
	ConsensusContextMaximumTransactionsInBlock() uint32
	ConsensusContextMinimumTransactionsInBlock() uint32
	ConsensusContextMinimalBlockTime() time.Duration
	FederationNodes(asOfBlock uint64) map[string]FederationNode
	ConsensusMinimumCommitteeSize() uint32
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
	NodePublicKey() primitives.Ed25519PublicKey
	NodePrivateKey() primitives.Ed25519PrivateKey
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
	NodePublicKey() primitives.Ed25519PublicKey
}

type GossipPeer interface {
	GossipPort() int
	GossipEndpoint() string
}
