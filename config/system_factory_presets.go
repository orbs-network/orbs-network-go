package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"path/filepath"
	"time"
)

// the default config is identical to ForProduction
// all other configs are variations from the production one
func defaultConfig() mutableNodeConfig {
	cfg := emptyConfig()

	cfg.SetActiveConsensusAlgo(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS)
	cfg.SetUint32(CONSENSUS_CONTEXT_PERCENTAGE_OF_NODES_REQUIRED_FOR_CONSENSUS, 100)
	cfg.SetUint32(VIRTUAL_CHAIN_ID, 42)
	cfg.SetUint32(BLOCK_TRACKER_GRACE_DISTANCE, 3)
	cfg.SetUint32(BLOCK_SYNC_BATCH_SIZE, 10000)
	cfg.SetDuration(BLOCK_SYNC_INTERVAL, 8*time.Second)
	cfg.SetDuration(BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT, 3*time.Second)
	cfg.SetDuration(BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT, 5*time.Second)
	cfg.SetDuration(BLOCK_TRANSACTION_RECEIPT_QUERY_GRACE_START, 5*time.Second)
	cfg.SetDuration(BLOCK_TRANSACTION_RECEIPT_QUERY_GRACE_END, 5*time.Second)
	cfg.SetDuration(BLOCK_TRANSACTION_RECEIPT_QUERY_EXPIRATION_WINDOW, 3*time.Minute)
	cfg.SetUint32(STATE_STORAGE_HISTORY_RETENTION_DISTANCE, 5)
	cfg.SetUint32(TRANSACTION_POOL_PENDING_POOL_SIZE_IN_BYTES, 20*1024*1024)
	cfg.SetDuration(TRANSACTION_POOL_TRANSACTION_EXPIRATION_WINDOW, 30*time.Minute)
	cfg.SetDuration(TRANSACTION_POOL_FUTURE_TIMESTAMP_GRACE_TIMEOUT, 5*time.Second)
	cfg.SetDuration(TRANSACTION_POOL_PENDING_POOL_CLEAR_EXPIRED_INTERVAL, 10*time.Second)
	cfg.SetDuration(TRANSACTION_POOL_COMMITTED_POOL_CLEAR_EXPIRED_INTERVAL, 30*time.Second)
	cfg.SetUint32(TRANSACTION_POOL_PROPAGATION_BATCH_SIZE, 5)
	cfg.SetDuration(TRANSACTION_POOL_PROPAGATION_BATCHING_TIMEOUT, 10*time.Millisecond)
	cfg.SetUint32(GOSSIP_LISTEN_PORT, 4400)
	cfg.SetString(PROCESSOR_ARTIFACT_PATH, filepath.Join(GetProjectSourceTmpPath(), "processor-artifacts"))
	cfg.SetDuration(METRICS_REPORT_INTERVAL, 30*time.Second)
	return cfg
}

// config for a production node (either main net or test net)
func ForProduction(processorArtifactPath string) mutableNodeConfig {
	cfg := defaultConfig()

	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 2*time.Second)
	cfg.SetDuration(CONSENSUS_CONTEXT_MINIMAL_BLOCK_DELAY, 1*time.Second) // this is the time between empty blocks when no transactions, need to be large so we don't close infinite blocks on idle
	cfg.SetUint32(CONSENSUS_CONTEXT_PERCENTAGE_OF_NODES_REQUIRED_FOR_CONSENSUS, 66)
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 100*time.Millisecond)
	cfg.SetDuration(GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, 1*time.Second)
	cfg.SetDuration(GOSSIP_NETWORK_TIMEOUT, 30*time.Second)
	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 30*time.Second)
	cfg.SetUint32(CONSENSUS_CONTEXT_MINIMUM_TRANSACTION_IN_BLOCK, 10)
	cfg.SetUint32(CONSENSUS_CONTEXT_MAXIMUM_TRANSACTION_IN_BLOCK, 100)
	cfg.SetUint32(TRANSACTION_POOL_PROPAGATION_BATCH_SIZE, 100)
	cfg.SetDuration(TRANSACTION_POOL_PROPAGATION_BATCHING_TIMEOUT, 100*time.Millisecond)
	if processorArtifactPath != "" {
		cfg.SetString(PROCESSOR_ARTIFACT_PATH, processorArtifactPath)
	}

	return cfg
}

// config for fast acceptance tests that run with in-memory adapters
func ForAcceptanceTests(
	federationNodes map[string]FederationNode,
	gossipPeers map[string]GossipPeer,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
	maxTxPerBlock uint32,
) mutableNodeConfig {
	cfg := defaultConfig()
	cfg.OverrideNodeSpecificValues(federationNodes, gossipPeers, 0, nodePublicKey, nodePrivateKey, constantConsensusLeader, activeConsensusAlgo)

	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 1*time.Millisecond)
	cfg.SetDuration(CONSENSUS_CONTEXT_MINIMAL_BLOCK_DELAY, 10*time.Millisecond)
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 50*time.Millisecond)
	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 300*time.Millisecond)
	cfg.SetUint32(CONSENSUS_CONTEXT_MINIMUM_TRANSACTION_IN_BLOCK, 1)
	cfg.SetUint32(CONSENSUS_CONTEXT_MAXIMUM_TRANSACTION_IN_BLOCK, maxTxPerBlock)
	cfg.SetUint32(TRANSACTION_POOL_PROPAGATION_BATCH_SIZE, 5)
	cfg.SetDuration(TRANSACTION_POOL_PROPAGATION_BATCHING_TIMEOUT, 3*time.Millisecond)
	cfg.SetUint32(BLOCK_SYNC_BATCH_SIZE, 5)
	cfg.SetDuration(BLOCK_SYNC_INTERVAL, 100*time.Millisecond)
	cfg.SetDuration(BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT, 15*time.Millisecond)
	cfg.SetDuration(BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT, 15*time.Millisecond)
	return cfg
}

// config for gamma dev network that runs with in-memory adapters except for contract compilation
func ForGamma(
	federationNodes map[string]FederationNode,
	gossipPeers map[string]GossipPeer,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
) mutableNodeConfig {
	cfg := defaultConfig()
	cfg.OverrideNodeSpecificValues(federationNodes, gossipPeers, 0, nodePublicKey, nodePrivateKey, constantConsensusLeader, activeConsensusAlgo)

	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 1000*time.Millisecond)
	cfg.SetDuration(CONSENSUS_CONTEXT_MINIMAL_BLOCK_DELAY, 500*time.Millisecond) // this is the time between empty blocks when no transactions, need to be large so we don't close infinite blocks on idle
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 100*time.Millisecond)
	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 10*time.Second)
	cfg.SetUint32(CONSENSUS_CONTEXT_MINIMUM_TRANSACTION_IN_BLOCK, 1)
	cfg.SetUint32(CONSENSUS_CONTEXT_MAXIMUM_TRANSACTION_IN_BLOCK, 100)
	cfg.SetUint32(BLOCK_SYNC_BATCH_SIZE, 5)
	cfg.SetDuration(BLOCK_SYNC_INTERVAL, 2500*time.Millisecond)
	cfg.SetDuration(BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT, 15*time.Millisecond)
	cfg.SetDuration(BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT, 15*time.Millisecond)
	return cfg
}
