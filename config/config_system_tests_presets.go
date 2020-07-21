// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"path/filepath"
	"time"
)

// config for end-to-end tests (very similar to production but slightly faster)
func ForE2E(
	httpAddress string,
	virtualChainId primitives.VirtualChainId,
	gossipListenPort int,
	nodeAddress primitives.NodeAddress,
	nodePrivateKey primitives.EcdsaSecp256K1PrivateKey,
	managementFilePath string,
	blockStorageDataDirPrefix string,
	processorArtifactPath string,
	ethereumEndpoint string,
	constantConsensusLeader primitives.NodeAddress,
	activeConsensusAlgo consensus.ConsensusAlgoType,
	experimentalExternalProcessorPluginPath string,
) NodeConfig {
	cfg := defaultProductionConfig()

	cfg.SetUint32(VIRTUAL_CHAIN_ID, uint32(virtualChainId))

	cfg.SetString(MANAGEMENT_FILE_PATH, managementFilePath)
	cfg.SetDuration(MANAGEMENT_CONSENSUS_GRACE_TIMEOUT, 0)
	cfg.SetDuration(COMMITTEE_VALIDITY_TIMEOUT, 100*365*24*time.Hour)

	// 2*slow_network_latency + avg_network_latency + 2*execution_time = 700ms
	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 700*time.Millisecond)
	// should be longer than tx_empty_block_time
	cfg.SetDuration(LEAN_HELIX_CONSENSUS_ROUND_TIMEOUT_INTERVAL, 2000*time.Millisecond)
	cfg.SetBool(LEAN_HELIX_SHOW_DEBUG, true)
	cfg.SetActiveConsensusAlgo(activeConsensusAlgo)
	cfg.SetBenchmarkConsensusConstantLeader(constantConsensusLeader)

	// longer than tx_empty_block_time and consensus round time
	cfg.SetDuration(BLOCK_SYNC_NO_COMMIT_INTERVAL, 3*time.Second)

	// 1MB blocks, 1KB per tx
	cfg.SetUint32(CONSENSUS_CONTEXT_MAXIMUM_TRANSACTIONS_IN_BLOCK, 1000)

	// max execution time (time validators allow until they get the executed block)
	cfg.SetDuration(CONSENSUS_CONTEXT_SYSTEM_TIMESTAMP_ALLOWED_JITTER, 30*time.Second)

	// scheduling hick-ups inside the node
	cfg.SetUint32(BLOCK_TRACKER_GRACE_DISTANCE, 5)
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 1*time.Second)

	// if above round time, we'll have leader changes when no traffic
	cfg.SetDuration(TRANSACTION_POOL_TIME_BETWEEN_EMPTY_BLOCKS, 1*time.Second) // this is the time between empty blocks when no transactions, need to be large so we don't close infinite blocks on idle
	cfg.SetDuration(PUBLIC_API_NODE_SYNC_WARNING_TIME, 30*time.Second)
	cfg.SetDuration(TRANSACTION_POOL_NODE_SYNC_REJECT_TIME, 1*time.Minute)
	// makes sync slower, 4*slow_network_latency
	cfg.SetDuration(BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT, 500*time.Millisecond)
	cfg.SetDuration(BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT, 2*time.Second)

	cfg.SetUint32(TRANSACTION_POOL_PROPAGATION_BATCH_SIZE, 1000)
	cfg.SetDuration(TRANSACTION_POOL_PROPAGATION_BATCHING_TIMEOUT, 50*time.Millisecond)

	cfg.SetUint32(GOSSIP_LISTEN_PORT, uint32(gossipListenPort))
	cfg.SetDuration(GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, 500*time.Millisecond)
	cfg.SetDuration(GOSSIP_NETWORK_TIMEOUT, 4*time.Second)
	cfg.SetDuration(GOSSIP_RECONNECT_INTERVAL, 500*time.Millisecond)

	cfg.SetString(ETHEREUM_ENDPOINT, ethereumEndpoint)
	cfg.SetDuration(ETHEREUM_FINALITY_TIME_COMPONENT, 1*time.Minute)
	cfg.SetUint32(ETHEREUM_FINALITY_BLOCKS_COMPONENT, 1)

	cfg.SetUint32(BLOCK_STORAGE_FILE_SYSTEM_MAX_BLOCK_SIZE_IN_BYTES, 64*1024*1024)
	cfg.SetString(BLOCK_STORAGE_FILE_SYSTEM_DATA_DIR, filepath.Join(blockStorageDataDirPrefix, nodeAddress.String()))

	cfg.SetBool(PROCESSOR_SANITIZE_DEPLOYED_CONTRACTS, true)
	if processorArtifactPath != "" {
		cfg.SetString(PROCESSOR_ARTIFACT_PATH, processorArtifactPath)
	}

	cfg.SetString(HTTP_ADDRESS, httpAddress)
	cfg.SetNodeAddress(nodeAddress)
	cfg.SetNodePrivateKey(nodePrivateKey)

	cfg.SetString(EXPERIMENTAL_EXTERNAL_PROCESSOR_PLUGIN_PATH, experimentalExternalProcessorPluginPath)

	return cfg
}

func ForAcceptanceTestNetwork(
	nodeAddress primitives.NodeAddress,
	privateKey primitives.EcdsaSecp256K1PrivateKey,
	constantConsensusLeader primitives.NodeAddress,
	activeConsensusAlgo consensus.ConsensusAlgoType,
	maxTxPerBlock uint32,
	requiredQuorumPercentage uint32,
	virtualChainId primitives.VirtualChainId,
	emptyBlockTime time.Duration,
	managementPollingInterval time.Duration,
	overrides ...NodeConfigKeyValue,
) mutableNodeConfig {
	cfg := defaultProductionConfig()

	if emptyBlockTime == 0 {
		emptyBlockTime = 50 * time.Millisecond
	}

	cfg.SetNodeAddress(nodeAddress)
	cfg.SetNodePrivateKey(privateKey)
	cfg.SetDuration(MANAGEMENT_POLLING_INTERVAL, managementPollingInterval)
	cfg.SetDuration(MANAGEMENT_CONSENSUS_GRACE_TIMEOUT, 0)
	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 50*time.Millisecond)
	cfg.SetDuration(LEAN_HELIX_CONSENSUS_ROUND_TIMEOUT_INTERVAL, 300*time.Millisecond)
	cfg.SetBool(LEAN_HELIX_SHOW_DEBUG, true)
	cfg.SetDuration(TRANSACTION_POOL_TIME_BETWEEN_EMPTY_BLOCKS, emptyBlockTime)
	cfg.SetUint32(BENCHMARK_CONSENSUS_REQUIRED_QUORUM_PERCENTAGE, requiredQuorumPercentage)
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 300*time.Millisecond)
	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 24*time.Hour) // ridiculously long timeout to reflect "forever"
	cfg.SetDuration(PUBLIC_API_NODE_SYNC_WARNING_TIME, 3000*time.Millisecond)
	cfg.SetUint32(CONSENSUS_CONTEXT_MAXIMUM_TRANSACTIONS_IN_BLOCK, maxTxPerBlock)
	cfg.SetUint32(TRANSACTION_POOL_PROPAGATION_BATCH_SIZE, 5)
	cfg.SetDuration(TRANSACTION_POOL_PROPAGATION_BATCHING_TIMEOUT, 3*time.Millisecond)
	cfg.SetUint32(BLOCK_SYNC_NUM_BLOCKS_IN_BATCH, 10)
	cfg.SetDuration(BLOCK_SYNC_NO_COMMIT_INTERVAL, 350*time.Millisecond)
	cfg.SetDuration(BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT, 15*time.Millisecond)
	cfg.SetDuration(BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT, 30*time.Millisecond)

	cfg.SetDuration(ETHEREUM_FINALITY_TIME_COMPONENT, 0*time.Millisecond)
	cfg.SetUint32(ETHEREUM_FINALITY_BLOCKS_COMPONENT, 0)
	cfg.SetUint32(VIRTUAL_CHAIN_ID, uint32(virtualChainId))

	cfg.SetBenchmarkConsensusConstantLeader(constantConsensusLeader)
	cfg.SetActiveConsensusAlgo(activeConsensusAlgo)

	cfg.Modify(overrides...)

	return cfg
}
