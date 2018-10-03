package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"path/filepath"
	"time"
)

func ForProduction(processorArtifactPath string) MutableNodeConfig {
	cfg := DefaultConfig()

	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 5*time.Second)
	cfg.SetDuration(CONSENSUS_CONTEXT_MINIMAL_BLOCK_DELAY, 1*time.Second) // this is the time between empty blocks when no transactions, need to be large so we don't close infinite blocks on idle
	cfg.SetUint32(CONSENSUS_CONTEXT_MAXIMUM_TRANSACTION_IN_BLOCK, 50)
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 100*time.Millisecond)
	cfg.SetDuration(GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, 5*time.Second)
	cfg.SetDuration(GOSSIP_NETWORK_TIMEOUT, 30*time.Second)
	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 30*time.Second)
	cfg.SetDuration(PUBLIC_API_TRANSACTION_STATUS_GRACE, 5*time.Second)
	cfg.SetUint32(CONSENSUS_CONTEXT_MINIMUM_TRANSACTION_IN_BLOCK, 10)
	cfg.SetUint32(CONSENSUS_CONTEXT_MAXIMUM_TRANSACTION_IN_BLOCK, 100)

	if processorArtifactPath != "" {
		cfg.SetString(PROCESSOR_ARTIFACT_PATH, processorArtifactPath)
	}

	return cfg
}

func ForAcceptanceTests(
	federationNodes map[string]FederationNode,
	gossipPeers map[string]GossipPeer,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
) MutableNodeConfig {
	cfg := DefaultConfig()
	cfg.OverrideNodeSpecificValues(federationNodes,
		gossipPeers,
		0,
		nodePublicKey,
		nodePrivateKey,
		constantConsensusLeader,
		activeConsensusAlgo)

	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 1*time.Millisecond)
	cfg.SetDuration(CONSENSUS_CONTEXT_MINIMAL_BLOCK_DELAY, 10*time.Millisecond)
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 5*time.Millisecond)
	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 100*time.Millisecond)
	cfg.SetDuration(PUBLIC_API_TRANSACTION_STATUS_GRACE, 5*time.Millisecond)
	cfg.SetUint32(CONSENSUS_CONTEXT_MINIMUM_TRANSACTION_IN_BLOCK, 1)
	cfg.SetUint32(CONSENSUS_CONTEXT_MAXIMUM_TRANSACTION_IN_BLOCK, 30)

	return cfg
}

func ForDevelopment(
	federationNodes map[string]FederationNode,
	gossipPeers map[string]GossipPeer,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
) MutableNodeConfig {
	cfg := DefaultConfig()
	cfg.OverrideNodeSpecificValues(federationNodes,
		gossipPeers,
		0,
		nodePublicKey,
		nodePrivateKey,
		constantConsensusLeader,
		activeConsensusAlgo)

	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 1000*time.Millisecond)
	cfg.SetDuration(CONSENSUS_CONTEXT_MINIMAL_BLOCK_DELAY, 500*time.Millisecond) // this is the time between empty blocks when no transactions, need to be large so we don't close infinite blocks on idle
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 100*time.Millisecond)
	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 10*time.Second)
	cfg.SetDuration(PUBLIC_API_TRANSACTION_STATUS_GRACE, 5*time.Millisecond)
	cfg.SetUint32(CONSENSUS_CONTEXT_MINIMUM_TRANSACTION_IN_BLOCK, 1)
	cfg.SetUint32(CONSENSUS_CONTEXT_MAXIMUM_TRANSACTION_IN_BLOCK, 100)

	return cfg
}

func EmptyConfig() MutableNodeConfig {
	return &config{
		kv: make(map[string]NodeConfigValue),
	}
}

func DefaultConfig() MutableNodeConfig {
	cfg := EmptyConfig()

	cfg.SetActiveConsensusAlgo(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS)

	cfg.SetUint32(VIRTUAL_CHAIN_ID, 42)

	cfg.SetUint32(BLOCK_TRACKER_GRACE_DISTANCE, 3)

	cfg.SetDuration(BLOCK_SYNC_BATCH_SIZE, 10000)
	cfg.SetDuration(BLOCK_SYNC_INTERVAL, 5*time.Second)
	cfg.SetDuration(BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT, 3*time.Second)
	cfg.SetDuration(BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT, 5*time.Second)

	cfg.SetDuration(BLOCK_TRANSACTION_RECEIPT_QUERY_GRACE_START, 5*time.Second)
	cfg.SetDuration(BLOCK_TRANSACTION_RECEIPT_QUERY_GRACE_END, 5*time.Second)
	cfg.SetDuration(BLOCK_TRANSACTION_RECEIPT_QUERY_EXPIRATION_WINDOW, 3*time.Minute)

	cfg.SetUint32(STATE_STORAGE_HISTORY_RETENTION_DISTANCE, 5)

	cfg.SetUint32(TRANSACTION_POOL_PENDING_POOL_SIZE_IN_BYTES, 20*1024*1024)
	cfg.SetDuration(TRANSACTION_POOL_TRANSACTION_EXPIRATION_WINDOW, 30*time.Minute)
	cfg.SetDuration(TRANSACTION_POOL_PENDING_POOL_CLEAR_EXPIRED_INTERVAL, 10*time.Second)
	cfg.SetDuration(TRANSACTION_POOL_COMMITTED_POOL_CLEAR_EXPIRED_INTERVAL, 30*time.Second)

	cfg.SetUint32(GOSSIP_LISTEN_PORT, 4400)

	cfg.SetString(PROCESSOR_ARTIFACT_PATH, filepath.Join(GetProjectSourceTmpPath(), "processor-artifacts"))

	return cfg
}

func (c *config) OverrideNodeSpecificValues(
	federationNodes map[string]FederationNode,
	gossipPeers map[string]GossipPeer,
	gossipListenPort uint16,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType) {

	c.SetFederationNodes(federationNodes)
	c.SetGossipPeers(gossipPeers)
	c.SetNodePublicKey(nodePublicKey)
	c.SetNodePrivateKey(nodePrivateKey)
	c.SetConstantConsensusLeader(constantConsensusLeader)
	c.SetActiveConsensusAlgo(activeConsensusAlgo)
	c.SetUint32(GOSSIP_LISTEN_PORT, uint32(gossipListenPort))
}

func (c *config) MergeWithFileConfig(source string) (MutableNodeConfig, error) {
	return NewFileConfig(c, source)
}
