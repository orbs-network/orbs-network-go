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

// all other configs are variations from the production one
func defaultProductionConfig() mutableNodeConfig {
	cfg := emptyConfig()

	cfg.SetUint32(VIRTUAL_CHAIN_ID, 42)
	cfg.SetUint32(GOSSIP_LISTEN_PORT, 4400)

	cfg.SetDuration(MANAGEMENT_POLLING_INTERVAL, 10*time.Second)
	cfg.SetUint32(MANAGEMENT_MAX_FILE_SIZE, 50*(1<<20)) // 50 MB
	cfg.SetDuration(MANAGEMENT_CONSENSUS_GRACE_TIMEOUT, 10*time.Minute)
	// for private consider changing this to 2^62 nanos (100 years) for PoS v2
	cfg.SetDuration(COMMITTEE_GRACE_PERIOD, 12*time.Hour)

	// 2*slow_network_latency + avg_network_latency + 2*execution_time \  + empty block time
	cfg.SetDuration(LEAN_HELIX_CONSENSUS_ROUND_TIMEOUT_INTERVAL, 14*time.Second)
	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 2*time.Second)

	cfg.SetUint32(LEAN_HELIX_CONSENSUS_MINIMUM_COMMITTEE_SIZE, 4)
	cfg.SetUint32(LEAN_HELIX_CONSENSUS_MAXIMUM_COMMITTEE_SIZE, 22)
	cfg.SetBool(LEAN_HELIX_SHOW_DEBUG, false)

	// if above round time, we'll have leader changes when no traffic
	cfg.SetDuration(TRANSACTION_POOL_TIME_BETWEEN_EMPTY_BLOCKS, 9*time.Second)

	cfg.SetUint32(BENCHMARK_CONSENSUS_REQUIRED_QUORUM_PERCENTAGE, 66)

	// 1MB blocks, 1KB per tx
	cfg.SetUint32(CONSENSUS_CONTEXT_MAXIMUM_TRANSACTIONS_IN_BLOCK, 1000)

	// max execution time (time validators allow until they get the executed block)
	cfg.SetDuration(CONSENSUS_CONTEXT_SYSTEM_TIMESTAMP_ALLOWED_JITTER, 60*time.Second)

	// have triggers transactions by default
	cfg.SetBool(CONSENSUS_CONTEXT_TRIGGERS_ENABLED, true)

	// scheduling hick-ups inside the node
	cfg.SetUint32(BLOCK_TRACKER_GRACE_DISTANCE, 5)
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 1*time.Second)

	// currently number of blocks held in memory
	cfg.SetUint32(BLOCK_SYNC_NUM_BLOCKS_IN_BATCH, 100)

	// 4*LEAN_HELIX_CONSENSUS_ROUND_TIMEOUT_INTERVAL, if below TRANSACTION_POOL_TIME_BETWEEN_EMPTY_BLOCKS we'll constantly have syncs
	cfg.SetDuration(BLOCK_SYNC_NO_COMMIT_INTERVAL, 18*time.Second)

	// makes sync slower, 4*slow_network_latency
	cfg.SetDuration(BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT, 1*time.Second)

	cfg.SetDuration(BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT, 5*time.Second)

	// have block sync use descending order of blocks from top
	cfg.SetBool(BLOCK_SYNC_DESCENDING_ENABLED, true)

	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 20*time.Second)

	// 5 empty blocks
	cfg.SetDuration(PUBLIC_API_NODE_SYNC_WARNING_TIME, 50*time.Second)

	cfg.SetDuration(BLOCK_STORAGE_TRANSACTION_RECEIPT_QUERY_TIMESTAMP_GRACE, 5*time.Second)

	cfg.SetUint32(STATE_STORAGE_HISTORY_SNAPSHOT_NUM, 5)
	cfg.SetUint32(TRANSACTION_POOL_PENDING_POOL_SIZE_IN_BYTES, 20*1024*1024)
	cfg.SetDuration(TRANSACTION_EXPIRATION_WINDOW, 30*time.Minute)

	// 2*PUBLIC_API_NODE_SYNC_WARNING_TIME
	cfg.SetDuration(TRANSACTION_POOL_NODE_SYNC_REJECT_TIME, 2*time.Minute)

	cfg.SetDuration(TRANSACTION_POOL_FUTURE_TIMESTAMP_GRACE_TIMEOUT, 1*time.Minute)
	cfg.SetDuration(TRANSACTION_POOL_PENDING_POOL_CLEAR_EXPIRED_INTERVAL, 10*time.Second)
	cfg.SetDuration(TRANSACTION_POOL_COMMITTED_POOL_CLEAR_EXPIRED_INTERVAL, 30*time.Second)

	cfg.SetUint32(TRANSACTION_POOL_PROPAGATION_BATCH_SIZE, 100)
	cfg.SetDuration(TRANSACTION_POOL_PROPAGATION_BATCHING_TIMEOUT, 100*time.Millisecond)

	cfg.SetDuration(GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, 1*time.Second)
	cfg.SetDuration(GOSSIP_RECONNECT_INTERVAL, 1*time.Second)
	cfg.SetDuration(GOSSIP_NETWORK_TIMEOUT, 30*time.Second)

	// 10 minutes + 60 blocks is about 25 minutes
	cfg.SetDuration(ETHEREUM_FINALITY_TIME_COMPONENT, 10*time.Minute)
	cfg.SetUint32(ETHEREUM_FINALITY_BLOCKS_COMPONENT, 60)

	cfg.SetBool(PROCESSOR_SANITIZE_DEPLOYED_CONTRACTS, true)
	cfg.SetBool(PROCESSOR_PERFORM_WARM_UP_COMPILATION, true)

	cfg.SetActiveConsensusAlgo(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS)
	cfg.SetString(ETHEREUM_ENDPOINT, "http://localhost:8545")
	cfg.SetString(PROCESSOR_ARTIFACT_PATH, filepath.Join(GetProjectSourceTmpPath(), "processor-artifacts"))
	cfg.SetString(BLOCK_STORAGE_FILE_SYSTEM_DATA_DIR, "/usr/local/var/orbs") // TODO V1 use build tags to replace with /var/lib/orbs for linux
	cfg.SetUint32(BLOCK_STORAGE_FILE_SYSTEM_MAX_BLOCK_SIZE_IN_BYTES, 64*1024*1024)

	cfg.SetDuration(LOGGER_FILE_TRUNCATION_INTERVAL, 24*time.Hour)
	cfg.SetBool(LOGGER_FULL_LOG, false)

	cfg.SetBool(PROFILING, false)
	cfg.SetString(HTTP_ADDRESS, ":8080")

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

// config for gamma dev network that runs with in-memory adapters except for contract compilation
func ForGamma(
	nodeAddress primitives.NodeAddress,
	privateKey primitives.EcdsaSecp256K1PrivateKey,
	constantConsensusLeader primitives.NodeAddress,
	serverAddress string,
	profiling bool,
	overrideJsonAsString string,

) mutableNodeConfig {
	cfg := defaultProductionConfig()

	cfg.SetNodeAddress(nodeAddress)
	cfg.SetNodePrivateKey(privateKey)
	cfg.SetBool(PROFILING, profiling)
	cfg.SetString(HTTP_ADDRESS, serverAddress)

	cfg.SetDuration(MANAGEMENT_CONSENSUS_GRACE_TIMEOUT, time.Hour) // needs to be >> from TRANSACTION_POOL_TIME_BETWEEN_EMPTY_BLOCKS
	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 100*time.Millisecond)
	cfg.SetDuration(TRANSACTION_POOL_TIME_BETWEEN_EMPTY_BLOCKS, 10*time.Minute)
	cfg.SetUint32(BENCHMARK_CONSENSUS_REQUIRED_QUORUM_PERCENTAGE, 100)
	cfg.SetDuration(LEAN_HELIX_CONSENSUS_ROUND_TIMEOUT_INTERVAL, 700*time.Millisecond)
	cfg.SetBool(LEAN_HELIX_SHOW_DEBUG, true)
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 100*time.Millisecond)
	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 10*time.Second)
	cfg.SetDuration(PUBLIC_API_NODE_SYNC_WARNING_TIME, 24*time.Hour)
	cfg.SetUint32(CONSENSUS_CONTEXT_MAXIMUM_TRANSACTIONS_IN_BLOCK, 10)
	cfg.SetBool(CONSENSUS_CONTEXT_TRIGGERS_ENABLED, false) // currently no reputation is needed in gamma
	cfg.SetUint32(TRANSACTION_POOL_PROPAGATION_BATCH_SIZE, 5)
	cfg.SetDuration(TRANSACTION_POOL_PROPAGATION_BATCHING_TIMEOUT, 10*time.Millisecond)
	cfg.SetDuration(TRANSACTION_POOL_NODE_SYNC_REJECT_TIME, 24*time.Hour)
	cfg.SetUint32(BLOCK_SYNC_NUM_BLOCKS_IN_BATCH, 2)
	cfg.SetDuration(BLOCK_SYNC_NO_COMMIT_INTERVAL, 1*time.Second)
	cfg.SetDuration(BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT, 100*time.Millisecond)
	cfg.SetDuration(BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT, 100*time.Millisecond)
	cfg.SetDuration(ETHEREUM_FINALITY_TIME_COMPONENT, 10*time.Second) // relevant for ganache
	cfg.SetUint32(ETHEREUM_FINALITY_BLOCKS_COMPONENT, 1)

	cfg.SetUint32(BLOCK_STORAGE_FILE_SYSTEM_MAX_BLOCK_SIZE_IN_BYTES, 64*1024*1024)
	cfg.SetString(ETHEREUM_ENDPOINT, "http://host.docker.internal:7545")

	cfg.SetBenchmarkConsensusConstantLeader(constantConsensusLeader)
	cfg.SetActiveConsensusAlgo(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS)

	cfg.SetString(PROCESSOR_ARTIFACT_PATH, filepath.Join(GetProjectSourceTmpPath(), "processor-artifacts"))
	// This is super important - The warmup compilation is disabled for gamma for a good reason since the plugins system
	// and Go's built-in race detector don't play along very well and we keep getting strange error when turning this on.
	// Around plugins the problem usually is a version mismatch for the warmup compilation for orbs-contract-sdk
	// The reason being the race detector is instrumenting the code of the package thus causing it to not be the same binary result
	// As the version of the package within the compiled plugin therefore the warmup compilation fails.
	cfg.SetBool(PROCESSOR_PERFORM_WARM_UP_COMPILATION, false)

	if overrideJsonAsString != "" {
		if err := modifyFromJson(cfg, overrideJsonAsString); err != nil {
			return nil
		}
	}

	return cfg
}
