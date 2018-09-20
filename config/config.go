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

	// benchmark consensus
	BenchmarkConsensusRetryInterval() time.Duration

	// block storage
	BlockSyncBatchSize() uint32
	BlockSyncInterval() time.Duration
	BlockSyncCollectResponseTimeout() time.Duration
	BlockTransactionReceiptQueryGraceStart() time.Duration
	BlockTransactionReceiptQueryGraceEnd() time.Duration
	BlockTransactionReceiptQueryExpirationWindow() time.Duration
	BlockSyncCollectChunksTimeout() time.Duration

	// state storage
	StateStorageHistoryRetentionDistance() uint32

	// block tracker
	BlockTrackerGraceDistance() uint32
	BlockTrackerGraceTimeout() time.Duration

	// consensus context
	ConsensusContextMinimalBlockDelay() time.Duration
	ConsensusContextMinimumTransactionsInBlock() uint32

	// transaction pool
	TransactionPoolPendingPoolSizeInBytes() uint32
	TransactionPoolTransactionExpirationWindow() time.Duration
	TransactionPoolFutureTimestampGraceTimeout() time.Duration
	TransactionPoolPendingPoolClearExpiredInterval() time.Duration
	TransactionPoolCommittedPoolClearExpiredInterval() time.Duration

	// gossip
	GossipListenPort() uint16
	GossipConnectionKeepAliveInterval() time.Duration
	GossipNetworkTimeout() time.Duration

	// public api
	SendTransactionTimeout() time.Duration
	GetTransactionStatusGrace() time.Duration

	// processor
	ProcessorArtifactPath() string
}

type MutableNodeConfig interface {
	NodeConfig

	Set(key string, value NodeConfigValue) MutableNodeConfig
	SetDuration(key string, value time.Duration) MutableNodeConfig
	SetUint32(key string, value uint32) MutableNodeConfig
	SetString(key string, value string) MutableNodeConfig
	SetFederationNodes(nodes map[string]FederationNode) MutableNodeConfig
	SetGossipPeers(peers map[string]GossipPeer) MutableNodeConfig
	SetNodePublicKey(key primitives.Ed25519PublicKey) MutableNodeConfig
	SetNodePrivateKey(key primitives.Ed25519PrivateKey) MutableNodeConfig

	SetConstantConsensusLeader(key primitives.Ed25519PublicKey) MutableNodeConfig
	SetActiveConsensusAlgo(algoType consensus.ConsensusAlgoType) MutableNodeConfig

	MergeWithFileConfig(source string) (MutableNodeConfig, error)
}

type FederationNode interface {
	NodePublicKey() primitives.Ed25519PublicKey
}

type GossipPeer interface {
	GossipPort() uint16
	GossipEndpoint() string
}
