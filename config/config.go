package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"time"
)

type NodeConfig interface {
	Set(key string, value NodeConfigValue) NodeConfig
	SetDuration(key string, value time.Duration) NodeConfig
	SetUint32(key string, value uint32) NodeConfig
	// TODO do we even need it
	SetNodePublicKey(key primitives.Ed25519PublicKey) NodeConfig

	NodePublicKey() primitives.Ed25519PublicKey
	NodePrivateKey() primitives.Ed25519PrivateKey
	NetworkSize(asOfBlock uint64) uint32
	FederationNodes(asOfBlock uint64) map[string]FederationNode
	QueryGraceTimeoutMillis() time.Duration

	// consensus
	ConstantConsensusLeader() primitives.Ed25519PublicKey
	ActiveConsensusAlgo() consensus.ConsensusAlgoType

	// benchmark consensus
	BenchmarkConsensusRoundRetryIntervalMillis() time.Duration

	// block storage
	BlockSyncCommitTimeoutMillis() time.Duration
	BlockTransactionReceiptQueryStartGraceSec() time.Duration
	BlockTransactionReceiptQueryEndGraceSec() time.Duration
	BlockTransactionReceiptQueryTransactionExpireSec() time.Duration

	// state storage
	StateHistoryRetentionInBlockHeights() uint32
	QuerySyncGraceBlockDist() uint32

	// consensus context
	BelowMinimalBlockDelayMillis() time.Duration
	MinimumTransactionsInBlock() uint32

	// transaction pool
	PendingPoolSizeInBytes() uint32
	TransactionExpirationWindowInSeconds() time.Duration
	FutureTimestampGraceInSeconds() time.Duration
	VirtualChainId() primitives.VirtualChainId
}

type FederationNode interface {
	NodePublicKey() primitives.Ed25519PublicKey
}
