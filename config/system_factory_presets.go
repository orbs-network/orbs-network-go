package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"path/filepath"
	"time"
)

// all other configs are variations from the production one
func defaultProductionConfig() mutableNodeConfig {
	cfg := emptyConfig()

	cfg.SetUint32(PROTOCOL_VERSION, 1)
	cfg.SetUint32(VIRTUAL_CHAIN_ID, 42)
	cfg.SetUint32(GOSSIP_LISTEN_PORT, 4400)
	cfg.SetActiveConsensusAlgo(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS)
	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 2*time.Second)
	cfg.SetDuration(CONSENSUS_CONTEXT_MINIMAL_BLOCK_TIME, 1*time.Second) // this is the time between empty blocks when no transactions, need to be large so we don't close infinite blocks on idle
	cfg.SetUint32(CONSENSUS_REQUIRED_QUORUM_PERCENTAGE, 66)
	cfg.SetUint32(CONSENSUS_CONTEXT_MINIMUM_TRANSACTIONS_IN_BLOCK, 10)
	cfg.SetUint32(CONSENSUS_CONTEXT_MAXIMUM_TRANSACTIONS_IN_BLOCK, 100)
	cfg.SetDuration(CONSENSUS_CONTEXT_SYSTEM_TIMESTAMP_ALLOWED_JITTER, 2*time.Second)
	cfg.SetUint32(CONSENSUS_MINIMUM_COMMITTEE_SIZE, 4)
	cfg.SetUint32(BLOCK_TRACKER_GRACE_DISTANCE, 3)
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 100*time.Millisecond)
	cfg.SetUint32(BLOCK_SYNC_BATCH_SIZE, 10000)
	cfg.SetDuration(BLOCK_SYNC_INTERVAL, 8*time.Second)
	cfg.SetDuration(BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT, 3*time.Second)
	cfg.SetDuration(BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT, 5*time.Second)
	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 30*time.Second)
	cfg.SetDuration(BLOCK_TRANSACTION_RECEIPT_QUERY_GRACE_START, 5*time.Second)
	cfg.SetDuration(BLOCK_TRANSACTION_RECEIPT_QUERY_GRACE_END, 5*time.Second)
	cfg.SetDuration(BLOCK_TRANSACTION_RECEIPT_QUERY_EXPIRATION_WINDOW, 3*time.Minute)
	cfg.SetUint32(STATE_STORAGE_HISTORY_SNAPSHOT_NUM, 5)
	cfg.SetUint32(TRANSACTION_POOL_PENDING_POOL_SIZE_IN_BYTES, 20*1024*1024)
	cfg.SetDuration(TRANSACTION_POOL_TRANSACTION_EXPIRATION_WINDOW, 30*time.Minute)
	cfg.SetDuration(TRANSACTION_POOL_FUTURE_TIMESTAMP_GRACE_TIMEOUT, 1*time.Minute)
	cfg.SetDuration(TRANSACTION_POOL_PENDING_POOL_CLEAR_EXPIRED_INTERVAL, 10*time.Second)
	cfg.SetDuration(TRANSACTION_POOL_COMMITTED_POOL_CLEAR_EXPIRED_INTERVAL, 30*time.Second)
	cfg.SetUint32(TRANSACTION_POOL_PROPAGATION_BATCH_SIZE, 100)
	cfg.SetDuration(TRANSACTION_POOL_PROPAGATION_BATCHING_TIMEOUT, 100*time.Millisecond)
	cfg.SetDuration(GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, 1*time.Second)
	cfg.SetDuration(GOSSIP_NETWORK_TIMEOUT, 30*time.Second)
	cfg.SetDuration(METRICS_REPORT_INTERVAL, 30*time.Second)
	cfg.SetString(PROCESSOR_ARTIFACT_PATH, filepath.Join(GetProjectSourceTmpPath(), "processor-artifacts"))
	return cfg
}

// config for a production node (either main net or test net)
func ForProduction(processorArtifactPath string) mutableNodeConfig {
	cfg := defaultProductionConfig()

	if processorArtifactPath != "" {
		cfg.SetString(PROCESSOR_ARTIFACT_PATH, processorArtifactPath)
	}
	return cfg
}

// config for end-to-end tests (very similar to production but slightly faster)
func ForE2E(
	processorArtifactPath string,
	federationNodes map[string]FederationNode,
	gossipPeers map[string]GossipPeer,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType) mutableNodeConfig {
	cfg := defaultProductionConfig()

	cfg.SetGossipPeers(gossipPeers)
	cfg.SetFederationNodes(federationNodes)
	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 250*time.Millisecond)
	cfg.SetDuration(CONSENSUS_CONTEXT_MINIMAL_BLOCK_TIME, 100*time.Millisecond) // this is the time between empty blocks when no transactions, need to be large so we don't close infinite blocks on idle
	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 10*time.Second)
	cfg.SetUint32(CONSENSUS_CONTEXT_MINIMUM_TRANSACTIONS_IN_BLOCK, 1)
	cfg.SetUint32(CONSENSUS_CONTEXT_MAXIMUM_TRANSACTIONS_IN_BLOCK, 100)
	cfg.SetUint32(TRANSACTION_POOL_PROPAGATION_BATCH_SIZE, 100)
	cfg.SetDuration(TRANSACTION_POOL_PROPAGATION_BATCHING_TIMEOUT, 50*time.Millisecond)
	cfg.SetDuration(BLOCK_SYNC_INTERVAL, 1000*time.Millisecond)
	cfg.SetActiveConsensusAlgo(activeConsensusAlgo)
	cfg.SetConstantConsensusLeader(constantConsensusLeader)

	if processorArtifactPath != "" {
		cfg.SetString(PROCESSOR_ARTIFACT_PATH, processorArtifactPath)
	}
	return cfg
}

func ForAcceptanceTestNetwork(
	federationNodes map[string]FederationNode,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
	maxTxPerBlock uint32,
	requiredQuorumPercentage uint32,
) mutableNodeConfig {
	cfg := defaultProductionConfig()
	cfg.SetFederationNodes(federationNodes)
	cfg.SetConstantConsensusLeader(constantConsensusLeader)
	cfg.SetActiveConsensusAlgo(activeConsensusAlgo)
	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 1*time.Millisecond)
	cfg.SetDuration(CONSENSUS_CONTEXT_MINIMAL_BLOCK_TIME, 10*time.Millisecond)
	cfg.SetUint32(CONSENSUS_REQUIRED_QUORUM_PERCENTAGE, requiredQuorumPercentage)
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 300*time.Millisecond)
	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 300*time.Millisecond)
	cfg.SetUint32(CONSENSUS_CONTEXT_MINIMUM_TRANSACTIONS_IN_BLOCK, 1)
	cfg.SetUint32(CONSENSUS_CONTEXT_MAXIMUM_TRANSACTIONS_IN_BLOCK, maxTxPerBlock)
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
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
) NodeConfig {
	cfg := defaultProductionConfig()
	cfg.SetFederationNodes(federationNodes)
	cfg.SetConstantConsensusLeader(constantConsensusLeader)
	cfg.SetActiveConsensusAlgo(activeConsensusAlgo)
	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 1000*time.Millisecond)
	cfg.SetDuration(CONSENSUS_CONTEXT_MINIMAL_BLOCK_TIME, 500*time.Millisecond) // this is the time between empty blocks when no transactions, need to be large so we don't close infinite blocks on idle
	cfg.SetUint32(CONSENSUS_REQUIRED_QUORUM_PERCENTAGE, 100)
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 100*time.Millisecond)
	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 10*time.Second)
	cfg.SetUint32(CONSENSUS_CONTEXT_MINIMUM_TRANSACTIONS_IN_BLOCK, 1)
	cfg.SetUint32(CONSENSUS_CONTEXT_MAXIMUM_TRANSACTIONS_IN_BLOCK, 100)
	cfg.SetUint32(TRANSACTION_POOL_PROPAGATION_BATCH_SIZE, 5)
	cfg.SetDuration(TRANSACTION_POOL_PROPAGATION_BATCHING_TIMEOUT, 10*time.Millisecond)
	cfg.SetUint32(BLOCK_SYNC_BATCH_SIZE, 5)
	cfg.SetDuration(BLOCK_SYNC_INTERVAL, 2500*time.Millisecond)
	cfg.SetDuration(BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT, 15*time.Millisecond)
	cfg.SetDuration(BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT, 15*time.Millisecond)
	return cfg.OverrideNodeSpecificValues(0, nodePublicKey, nodePrivateKey)
}
