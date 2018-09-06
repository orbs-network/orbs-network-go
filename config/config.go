package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"time"
)

type NodeConfig interface {
	// setters (for creation)
	Set(key string, value NodeConfigValue) NodeConfig
	SetDuration(key string, value time.Duration) NodeConfig
	SetUint32(key string, value uint32) NodeConfig
	SetFederationNodes(map[string]FederationNode) NodeConfig
	SetNodePublicKey(key primitives.Ed25519PublicKey) NodeConfig
	SetNodePrivateKey(key primitives.Ed25519PrivateKey) NodeConfig

	// shared
	VirtualChainId() primitives.VirtualChainId
	NodePublicKey() primitives.Ed25519PublicKey
	NodePrivateKey() primitives.Ed25519PrivateKey
	NetworkSize(asOfBlock uint64) uint32
	FederationNodes(asOfBlock uint64) map[string]FederationNode

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
	GossipConnectionKeepAliveInterval() time.Duration
	GossipNetworkTimeout() time.Duration

	// public api
	SendTransactionTimeout() time.Duration
	GetTransactionStatusGrace() time.Duration
}

type FederationNode interface {
	NodePublicKey() primitives.Ed25519PublicKey
	GossipPort() uint16
	GossipEndpoint() string
}
