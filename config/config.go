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
	VirtualChainId() primitives.VirtualChainId
	FederationNodes(asOfBlock uint64) map[string]FederationNode
	QueryGraceTimeoutMillis() uint64

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
	StateHistoryRetentionInBlockHeights() uint16
	QuerySyncGraceBlockDist() uint16

	// consensus context
	BelowMinimalBlockDelayMillis() uint32
	MinimumTransactionsInBlock() int

	// transaction pool
	PendingPoolSizeInBytes() uint32
	TransactionExpirationWindow() time.Duration
	FutureTimestampGraceInSeconds() uint32
	PendingPoolClearExpiredInterval() time.Duration
	CommittedPoolClearExpiredInterval() time.Duration
}

type FederationNode interface {
	NodePublicKey() primitives.Ed25519PublicKey
}
