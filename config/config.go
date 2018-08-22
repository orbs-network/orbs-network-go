package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"time"
)

type NodeConfig interface {
	NodePublicKey() primitives.Ed25519PublicKey
	NodePrivateKey() primitives.Ed25519PrivateKey
	NetworkSize(asOfBlock uint64) uint32
	FederationNodes(asOfBlock uint64) map[string]FederationNode
	QueryGraceTimeoutMillis() time.Duration

	// consensus
	ConstantConsensusLeader() primitives.Ed25519PublicKey
	ActiveConsensusAlgo() consensus.ConsensusAlgoType

	// benchmark consensus
	BenchmarkConsensusRoundRetryIntervalMillis() uint32

	// block storage
	BlockSyncCommitTimeoutMillis() time.Duration
	BlockTransactionReceiptQueryStartGraceSec() time.Duration
	BlockTransactionReceiptQueryEndGraceSec() time.Duration
	BlockTransactionReceiptQueryTransactionExpireSec() time.Duration

	// state storage
	StateHistoryRetentionInBlockHeights() uint32
	QuerySyncGraceBlockDist() uint32

	// consensus context
	BelowMinimalBlockDelayMillis() uint32
	MinimumTransactionsInBlock() uint32

	// transaction pool
	PendingPoolSizeInBytes() uint32
	TransactionExpirationWindowInSeconds() uint32
	FutureTimestampGraceInSeconds() uint32
	VirtualChainId() primitives.VirtualChainId
}

type FederationNode interface {
	NodePublicKey() primitives.Ed25519PublicKey
}
