package config

import (
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"time"
)

func ForDirectTransportTests(gossipPeers map[string]GossipPeer, keepAliveInterval time.Duration) GossipTransportConfig {
	cfg := emptyConfig()
	cfg.SetNodeAddress(testKeys.EcdsaSecp256K1KeyPairForTests(0).NodeAddress())
	cfg.SetGossipPeers(gossipPeers)

	cfg.SetDuration(GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, 20*time.Millisecond)
	cfg.SetDuration(GOSSIP_NETWORK_TIMEOUT, keepAliveInterval)
	return cfg
}

func ForGossipAdapterTests(nodeAddress primitives.NodeAddress, gossipListenPort int, gossipPeers map[string]GossipPeer) GossipTransportConfig {
	cfg := emptyConfig()
	cfg.SetNodeAddress(nodeAddress)
	cfg.SetGossipPeers(gossipPeers)

	cfg.SetUint32(GOSSIP_LISTEN_PORT, uint32(gossipListenPort))
	cfg.SetDuration(GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, 20*time.Millisecond)
	cfg.SetDuration(GOSSIP_NETWORK_TIMEOUT, 1*time.Second)
	return cfg
}

func ForConsensusContextTests(federationNodes map[string]FederationNode) ConsensusContextConfig {
	cfg := emptyConfig()

	cfg.SetUint32(PROTOCOL_VERSION, 1)
	cfg.SetBool(LEAN_HELIX_SHOW_DEBUG, true)
	cfg.SetUint32(VIRTUAL_CHAIN_ID, 42)
	cfg.SetUint32(NETWORK_TYPE, uint32(protocol.NETWORK_TYPE_TEST_NET))
	cfg.SetUint32(LEAN_HELIX_CONSENSUS_MINIMUM_COMMITTEE_SIZE, 4)
	cfg.SetDuration(CONSENSUS_CONTEXT_SYSTEM_TIMESTAMP_ALLOWED_JITTER, 2*time.Second)
	if federationNodes != nil {
		cfg.SetFederationNodes(federationNodes)
	}
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

func ForTransactionPoolTests(sizeLimit uint32, keyPair *testKeys.TestEcdsaSecp256K1KeyPair) TransactionPoolConfig {
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
	cfg.SetDuration(TRANSACTION_POOL_TIME_BETWEEN_EMPTY_BLOCKS, 100*time.Millisecond)
	return cfg
}
