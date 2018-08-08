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
	BenchmarkConsensusRoundRetryIntervalMillisec() uint32

	// block storage
	BlockSyncCommitTimeoutMillisec() time.Duration

	// state storage
	StateHistoryRetentionInBlockHeights() uint64
}

type FederationNode interface {
	NodePublicKey() primitives.Ed25519PublicKey
}
