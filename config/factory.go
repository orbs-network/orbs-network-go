package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"time"
)

func ForProduction(
	federationNodes map[string]FederationNode,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
) NodeConfig {

	cfg := newHardCodedConfig(
		federationNodes,
		nodePublicKey,
		nodePrivateKey,
		constantConsensusLeader,
		activeConsensusAlgo)

	cfg.SetUint32(CONSENSUS_CONTEXT_MINIMUM_TRANSACTION_IN_BLOCK, uint32(1))
	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 2000*time.Millisecond)
	cfg.SetDuration(CONSENSUS_CONTEXT_MINIMAL_BLOCK_DELAY, 20*time.Millisecond) // longer than in acceptance test because otherwise e2e flakes. TODO figure out why
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 100*time.Millisecond)
	cfg.SetDuration(GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, 5*time.Second)
	cfg.SetDuration(GOSSIP_NETWORK_TIMEOUT, 30*time.Second)

	return cfg
}

func ForAcceptanceTests(
	federationNodes map[string]FederationNode,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
) NodeConfig {

	cfg := newHardCodedConfig(
		federationNodes,
		nodePublicKey,
		nodePrivateKey,
		constantConsensusLeader,
		activeConsensusAlgo)

	cfg.SetUint32(CONSENSUS_CONTEXT_MINIMUM_TRANSACTION_IN_BLOCK, uint32(1))
	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 1*time.Millisecond)
	cfg.SetDuration(CONSENSUS_CONTEXT_MINIMAL_BLOCK_DELAY, 1*time.Millisecond)
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 5*time.Millisecond)
	cfg.SetDuration(GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, 1*time.Millisecond)
	cfg.SetDuration(GOSSIP_NETWORK_TIMEOUT, 1*time.Second)
	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 30*time.Millisecond)
	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 30*time.Millisecond)
	cfg.SetDuration(PUBLIC_API_TRANSACTION_STATUS_GRACE, 5*time.Millisecond)

	return cfg
}

func EmptyConfig() NodeConfig {
	return &config{
		kv: make(map[string]NodeConfigValue),
	}
}
