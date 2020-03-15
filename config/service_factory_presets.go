// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package config

import (
	topologyProviderAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"time"
)

func ForDirectTransportTests(nodeAddress primitives.NodeAddress, gossipPeers topologyProviderAdapter.GossipPeers, keepAliveInterval time.Duration, networkTimeout time.Duration) GossipTransportConfig {
	cfg := emptyConfig()
	cfg.SetNodeAddress(nodeAddress)
	cfg.SetGossipPeers(gossipPeers)

	cfg.SetDuration(GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, keepAliveInterval)
	cfg.SetDuration(GOSSIP_NETWORK_TIMEOUT, networkTimeout)
	cfg.SetDuration(GOSSIP_RECONNECT_INTERVAL, 20*time.Millisecond)
	cfg.SetDuration(MANAGEMENT_UPDATE_INTERVAL, 100*time.Millisecond)

	return cfg
}

func ForGossipAdapterTests(nodeAddress primitives.NodeAddress) GossipTransportConfig {
	cfg := emptyConfig()
	cfg.SetNodeAddress(nodeAddress)
	cfg.SetGossipPeers(make(topologyProviderAdapter.GossipPeers))

	cfg.SetUint32(GOSSIP_LISTEN_PORT, uint32(0))
	cfg.SetDuration(GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, 20*time.Millisecond)
	cfg.SetDuration(GOSSIP_NETWORK_TIMEOUT, 1*time.Second)
	cfg.SetDuration(GOSSIP_RECONNECT_INTERVAL, 20*time.Millisecond)
	cfg.SetDuration(MANAGEMENT_UPDATE_INTERVAL, 10*time.Second)

	return cfg
}

func ForConsensusContextTests(triggersEnabled bool) ConsensusContextConfig {
	cfg := emptyConfig()

	cfg.SetUint32(PROTOCOL_VERSION, 1)
	cfg.SetBool(LEAN_HELIX_SHOW_DEBUG, true)
	cfg.SetUint32(VIRTUAL_CHAIN_ID, 42)
	cfg.SetUint32(NETWORK_TYPE, uint32(protocol.NETWORK_TYPE_TEST_NET))
	cfg.SetUint32(LEAN_HELIX_CONSENSUS_MINIMUM_COMMITTEE_SIZE, 4)
	cfg.SetDuration(CONSENSUS_CONTEXT_SYSTEM_TIMESTAMP_ALLOWED_JITTER, 2*time.Second)
	cfg.SetBool(CONSENSUS_CONTEXT_TRIGGERS_ENABLED, triggersEnabled)

	return cfg
}

func ForPublicApiTests(virtualChain uint32, txTimeout time.Duration, outOfSyncWarningTime time.Duration) PublicApiConfig {
	cfg := emptyConfig()

	cfg.SetUint32(VIRTUAL_CHAIN_ID, virtualChain)
	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, txTimeout)
	cfg.SetDuration(PUBLIC_API_NODE_SYNC_WARNING_TIME, outOfSyncWarningTime)
	return cfg
}

func ForStateStorageTest(numOfStateRevisionsToRetain uint32, graceBlockDiff uint32, graceTimeoutMillis uint64) StateStorageConfig {
	cfg := emptyConfig()

	cfg.SetUint32(STATE_STORAGE_HISTORY_SNAPSHOT_NUM, numOfStateRevisionsToRetain)
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, time.Duration(graceTimeoutMillis)*time.Millisecond)
	cfg.SetUint32(BLOCK_TRACKER_GRACE_DISTANCE, graceBlockDiff)
	return cfg
}

func ForTransactionPoolTests(sizeLimit uint32, keyPair *testKeys.TestEcdsaSecp256K1KeyPair, timeBetweenEmptyBlocks time.Duration) TransactionPoolConfigForTests {
	cfg := emptyConfig()
	cfg.SetNodeAddress(keyPair.NodeAddress())
	cfg.SetNodePrivateKey(keyPair.PrivateKey())

	cfg.SetUint32(VIRTUAL_CHAIN_ID, 42)
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 100*time.Millisecond)
	cfg.SetUint32(BLOCK_TRACKER_GRACE_DISTANCE, 5)
	cfg.SetUint32(TRANSACTION_POOL_PENDING_POOL_SIZE_IN_BYTES, sizeLimit)
	cfg.SetDuration(TRANSACTION_POOL_NODE_SYNC_REJECT_TIME, 2*time.Minute)
	cfg.SetDuration(TRANSACTION_EXPIRATION_WINDOW, 30*time.Minute)
	cfg.SetDuration(TRANSACTION_POOL_FUTURE_TIMESTAMP_GRACE_TIMEOUT, 3*time.Minute)
	cfg.SetDuration(TRANSACTION_POOL_PENDING_POOL_CLEAR_EXPIRED_INTERVAL, 3*time.Second)
	cfg.SetDuration(TRANSACTION_POOL_COMMITTED_POOL_CLEAR_EXPIRED_INTERVAL, 3*time.Second)
	cfg.SetUint32(TRANSACTION_POOL_PROPAGATION_BATCH_SIZE, 1)
	cfg.SetDuration(TRANSACTION_POOL_PROPAGATION_BATCHING_TIMEOUT, 50*time.Millisecond)
	cfg.SetDuration(TRANSACTION_POOL_TIME_BETWEEN_EMPTY_BLOCKS, timeBetweenEmptyBlocks)
	return cfg
}

func ForLeanHelixConsensusTests(keyPair *testKeys.TestEcdsaSecp256K1KeyPair, auditBlocksYoungerThan time.Duration, consensusRoundTimeoutInterval time.Duration) LeanHelixConsensusConfigForTests {
	cfg := emptyConfig()
	cfg.SetNodeAddress(keyPair.NodeAddress())
	cfg.SetNodePrivateKey(keyPair.PrivateKey())

	cfg.SetActiveConsensusAlgo(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX)
	cfg.SetDuration(LEAN_HELIX_CONSENSUS_ROUND_TIMEOUT_INTERVAL, consensusRoundTimeoutInterval)
	cfg.SetUint32(LEAN_HELIX_CONSENSUS_MAXIMUM_COMMITTEE_SIZE, 22)
	cfg.SetBool(LEAN_HELIX_SHOW_DEBUG, true)
	cfg.SetUint32(VIRTUAL_CHAIN_ID, 42)
	cfg.SetUint32(NETWORK_TYPE, uint32(protocol.NETWORK_TYPE_TEST_NET))

	cfg.SetDuration(INTER_NODE_SYNC_AUDIT_BLOCKS_YOUNGER_THAN, auditBlocksYoungerThan)

	return cfg
}

func ForBenchmarkConsensusTests(keyPair *testKeys.TestEcdsaSecp256K1KeyPair, leaderKeyPair *testKeys.TestEcdsaSecp256K1KeyPair, validators map[string]ValidatorNode) NodeConfig {
	cfg := emptyConfig()
	cfg.SetGenesisValidatorNodes(validators)
	cfg.SetBenchmarkConsensusConstantLeader(leaderKeyPair.NodeAddress())
	cfg.SetActiveConsensusAlgo(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS)
	cfg.SetUint32(CONSENSUS_CONTEXT_MAXIMUM_TRANSACTIONS_IN_BLOCK, 1)
	cfg.SetUint32(VIRTUAL_CHAIN_ID, 42)
	cfg.SetDuration(BENCHMARK_CONSENSUS_RETRY_INTERVAL, 5*time.Millisecond)
	cfg.SetUint32(BENCHMARK_CONSENSUS_REQUIRED_QUORUM_PERCENTAGE, 66)
	cfg.SetNodeAddress(keyPair.NodeAddress())
	cfg.SetNodePrivateKey(keyPair.PrivateKey())

	return cfg
}

func ForNativeProcessorTests(id primitives.VirtualChainId) NativeProcessorConfig {
	cfg := emptyConfig()
	cfg.SetUint32(VIRTUAL_CHAIN_ID, uint32(id))
	return cfg
}
