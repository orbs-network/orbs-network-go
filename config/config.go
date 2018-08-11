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
	StateHistoryRetentionInBlockHeights() uint64

	// consensus context
	BelowMinimalBlockDelayMillis() uint32
	MinimumTransactionsInBlock() int
}

type FederationNode interface {
	NodePublicKey() primitives.Ed25519PublicKey
}
