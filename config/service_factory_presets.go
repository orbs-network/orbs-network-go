package config

import (
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"time"
)

func ForDirectTransportTests(gossipPeers map[string]GossipPeer) GossipTransportConfig {
	cfg := emptyConfig()
	cfg.SetNodePublicKey(testKeys.Ed25519KeyPairForTests(0).PublicKey())
	cfg.SetGossipPeers(gossipPeers)

	cfg.SetDuration(GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, 20*time.Millisecond)
	cfg.SetDuration(GOSSIP_NETWORK_TIMEOUT, 20*time.Millisecond)
	return cfg
}

func ForGossipAdapterTests(publicKey primitives.Ed25519PublicKey, gossipListenPort int, gossipPeers map[string]GossipPeer) GossipTransportConfig {
	cfg := emptyConfig()
	cfg.SetNodePublicKey(publicKey)
	cfg.SetGossipPeers(gossipPeers)

	cfg.SetUint32(GOSSIP_LISTEN_PORT, uint32(gossipListenPort))
	cfg.SetDuration(GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, 20*time.Millisecond)
	cfg.SetDuration(GOSSIP_NETWORK_TIMEOUT, 1*time.Second)
	return cfg
}

func ForConsensusContextTests(federationNodes map[string]FederationNode) ConsensusContextConfig {
	cfg := emptyConfig()

	cfg.SetUint32(PROTOCOL_VERSION, 1)
	cfg.SetUint32(VIRTUAL_CHAIN_ID, 42)
	cfg.SetDuration(CONSENSUS_CONTEXT_MINIMAL_BLOCK_TIME, 1*time.Millisecond)
	cfg.SetUint32(CONSENSUS_CONTEXT_MINIMUM_TRANSACTIONS_IN_BLOCK, 2)
	cfg.SetUint32(CONSENSUS_MINIMUM_COMMITTEE_SIZE, 4)
	cfg.SetDuration(CONSENSUS_CONTEXT_SYSTEM_TIMESTAMP_ALLOWED_JITTER, 2*time.Second)
	if federationNodes != nil {
		cfg.SetFederationNodes(federationNodes)
	}
	return cfg
}

func ForPublicApiTests(virtualChain uint32, txTimeout time.Duration) PublicApiConfig {
	cfg := emptyConfig()

	cfg.SetUint32(VIRTUAL_CHAIN_ID, virtualChain)
	cfg.SetDuration(PUBLIC_API_SEND_TRANSACTION_TIMEOUT, txTimeout)
	return cfg
}

func ForStateStorageTest(numOfStateRevisionsToRetain uint32, graceBlockDiff uint32, graceTimeoutMillis uint64) StateStorageConfig {
	cfg := emptyConfig()

	cfg.SetUint32(STATE_STORAGE_HISTORY_SNAPSHOT_NUM, numOfStateRevisionsToRetain)
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, time.Duration(graceTimeoutMillis)*time.Millisecond)
	cfg.SetUint32(BLOCK_TRACKER_GRACE_DISTANCE, graceBlockDiff)
	return cfg
}

func ForTransactionPoolTests(sizeLimit uint32, keyPair *keys.Ed25519KeyPair) TransactionPoolConfig {
	cfg := emptyConfig()
	cfg.SetNodePublicKey(keyPair.PublicKey())
	cfg.SetNodePrivateKey(keyPair.PrivateKey())

	cfg.SetUint32(VIRTUAL_CHAIN_ID, 42)
	cfg.SetDuration(BLOCK_TRACKER_GRACE_TIMEOUT, 100*time.Millisecond)
	cfg.SetUint32(BLOCK_TRACKER_GRACE_DISTANCE, 5)
	cfg.SetUint32(TRANSACTION_POOL_PENDING_POOL_SIZE_IN_BYTES, sizeLimit)
	cfg.SetDuration(TRANSACTION_POOL_TRANSACTION_EXPIRATION_WINDOW, 30*time.Minute)
	cfg.SetDuration(TRANSACTION_POOL_FUTURE_TIMESTAMP_GRACE_TIMEOUT, 3*time.Minute)
	cfg.SetDuration(TRANSACTION_POOL_PENDING_POOL_CLEAR_EXPIRED_INTERVAL, 10*time.Millisecond)
	cfg.SetDuration(TRANSACTION_POOL_COMMITTED_POOL_CLEAR_EXPIRED_INTERVAL, 30*time.Millisecond)
	cfg.SetUint32(TRANSACTION_POOL_PROPAGATION_BATCH_SIZE, 1)
	cfg.SetDuration(TRANSACTION_POOL_PROPAGATION_BATCHING_TIMEOUT, 50*time.Millisecond)
	return cfg
}
