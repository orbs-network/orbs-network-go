package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"time"
)

func ForProduction(
	federationNodes map[string]FederationNode,
	gossipPeers map[string]GossipPeer,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	gossipListenPort uint16,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
	processorArtifactPath string,
) NodeConfig {

	cfg := newHardCodedConfig(
		federationNodes,
		gossipPeers,
		nodePublicKey,
		nodePrivateKey,
		constantConsensusLeader,
		activeConsensusAlgo,
		processorArtifactPath,
	)

	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 5*time.Second)
	cfg.SetDuration(CONSENSUS_CONTEXT_MINIMAL_BLOCK_DELAY, 1*time.Second) // this is the time between empty blocks when no transactions, need to be large so we don't close infinite blocks on idle
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 100*time.Millisecond)
	cfg.SetUint32(GOSSIP_LISTEN_PORT, uint32(gossipListenPort))
	cfg.SetDuration(GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, 5*time.Second)
	cfg.SetDuration(GOSSIP_NETWORK_TIMEOUT, 30*time.Second)
	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 30*time.Second)
	cfg.SetDuration(PUBLIC_API_TRANSACTION_STATUS_GRACE, 5*time.Second)

	return cfg
}

func ForAcceptanceTests(
	federationNodes map[string]FederationNode,
	gossipPeers map[string]GossipPeer,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
) NodeConfig {

	cfg := newHardCodedConfig(
		federationNodes,
		gossipPeers,
		nodePublicKey,
		nodePrivateKey,
		constantConsensusLeader,
		activeConsensusAlgo,
		"", // default
	)

	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 1*time.Millisecond)
	cfg.SetDuration(CONSENSUS_CONTEXT_MINIMAL_BLOCK_DELAY, 10*time.Millisecond)
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 5*time.Millisecond)
	cfg.SetUint32(GOSSIP_LISTEN_PORT, 0)
	cfg.SetDuration(GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, 1*time.Millisecond)
	cfg.SetDuration(GOSSIP_NETWORK_TIMEOUT, 1*time.Second)
	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 30*time.Millisecond)
	cfg.SetDuration(PUBLIC_API_TRANSACTION_STATUS_GRACE, 5*time.Millisecond)

	return cfg
}

func ForDevelopment(
	federationNodes map[string]FederationNode,
	gossipPeers map[string]GossipPeer,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
) NodeConfig {

	cfg := newHardCodedConfig(
		federationNodes,
		gossipPeers,
		nodePublicKey,
		nodePrivateKey,
		constantConsensusLeader,
		activeConsensusAlgo,
		"", // default
	)

	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 1000*time.Millisecond)
	cfg.SetDuration(CONSENSUS_CONTEXT_MINIMAL_BLOCK_DELAY, 500*time.Millisecond) // this is the time between empty blocks when no transactions, need to be large so we don't close infinite blocks on idle
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 100*time.Millisecond)
	cfg.SetUint32(GOSSIP_LISTEN_PORT, 0)
	cfg.SetDuration(GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, 1*time.Millisecond)
	cfg.SetDuration(GOSSIP_NETWORK_TIMEOUT, 1*time.Second)
	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 10*time.Second)
	cfg.SetDuration(PUBLIC_API_TRANSACTION_STATUS_GRACE, 5*time.Millisecond)

	return cfg
}

func EmptyConfig() NodeConfig {
	return &config{
		kv: make(map[string]NodeConfigValue),
	}
}

func (c *config) MergeWithFileConfig(source string) (NodeConfig, error) {
	return NewFileConfig(c, source)
}
