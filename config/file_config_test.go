package config

import (
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const FILE_CONFIG_CONTENTS = `
{
	"block-sync-batch-size": 999,
	"block-sync-collect-response-timeout": "10m",
	"node-public-key": "dfc06c5be24a67adee80b35ab4f147bb1a35c55ff85eda69f40ef827bddec173",
	"node-private-key": "93e919986a22477fda016789cca30cb841a135650938714f85f0000a65076bd4dfc06c5be24a67adee80b35ab4f147bb1a35c55ff85eda69f40ef827bddec173",
	"constant-consensus-leader": "92d469d7c004cc0b24a192d9457836bf38effa27536627ef60718b00b0f33152",
	"active-consensus-algo": 999,
	"gossip-port": 4500,
	"federation-nodes": [
		{"Key":"dfc06c5be24a67adee80b35ab4f147bb1a35c55ff85eda69f40ef827bddec173","IP":"192.168.199.2","Port":4400},
		{"Key":"92d469d7c004cc0b24a192d9457836bf38effa27536627ef60718b00b0f33152","IP":"192.168.199.3","Port":4400},
		{"Key":"a899b318e65915aa2de02841eeb72fe51fddad96014b73800ca788a547f8cce0","IP":"192.168.199.4","Port":4400}
	]
}
`

const EMPTY_FILE_CONFIG = `{}`

func TestFileConfigConstructor(t *testing.T) {
	cfg, err := NewEmptyFileConfig(EMPTY_FILE_CONFIG)

	require.NotNil(t, cfg)
	require.NoError(t, err)
}

const FILE_CONFIG_WITH_BLOCK_SYNC = `{"block-sync-batch-size": 999}`

func TestFileConfigSetUint32(t *testing.T) {
	cfg, err := NewEmptyFileConfig(FILE_CONFIG_WITH_BLOCK_SYNC)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, 999, cfg.BlockSyncBatchSize())
}

const FILE_CONFIG_WITH_BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT = `{"block-sync-collect-response-timeout": "10m"}`

func TestFileConfigSetDuration(t *testing.T) {
	cfg, err := NewEmptyFileConfig(FILE_CONFIG_WITH_BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, 10*time.Minute, cfg.BlockSyncCollectResponseTimeout())
}

const FILE_CONFIG_WITH_PUBLIC_KEY = `{"node-public-key": "dfc06c5be24a67adee80b35ab4f147bb1a35c55ff85eda69f40ef827bddec173"}`

func TestSetNodePublicKey(t *testing.T) {
	cfg, err := NewEmptyFileConfig(FILE_CONFIG_WITH_PUBLIC_KEY)

	keyPair := keys.Ed25519KeyPairForTests(0)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, keyPair.PublicKey(), cfg.NodePublicKey())
}

const FILE_CONFIG_WITH_PRIVATE_KEY = `{"node-private-key": "93e919986a22477fda016789cca30cb841a135650938714f85f0000a65076bd4dfc06c5be24a67adee80b35ab4f147bb1a35c55ff85eda69f40ef827bddec173"}`

func TestSetNodePrivateKey(t *testing.T) {
	cfg, err := NewEmptyFileConfig(FILE_CONFIG_WITH_PRIVATE_KEY)

	keyPair := keys.Ed25519KeyPairForTests(0)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, keyPair.PrivateKey(), cfg.NodePrivateKey())
}

const FILE_CONFIG_WITH_CONSTANT_CONSENSUS_LEADER = `{"constant-consensus-leader": "92d469d7c004cc0b24a192d9457836bf38effa27536627ef60718b00b0f33152"}`

func TestSetConstantConsensusLeader(t *testing.T) {
	cfg, err := NewEmptyFileConfig(FILE_CONFIG_WITH_CONSTANT_CONSENSUS_LEADER)

	keyPair := keys.Ed25519KeyPairForTests(1)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, keyPair.PublicKey(), cfg.ConstantConsensusLeader())
}

const FILE_CONFIG_WITH_ACTIVE_CONSENSUS_ALGO = `{"active-consensus-algo": 999}`

func TestSetActiveConsensusAlgo(t *testing.T) {
	cfg, err := NewEmptyFileConfig(FILE_CONFIG_WITH_ACTIVE_CONSENSUS_ALGO)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, 999, cfg.ActiveConsensusAlgo())
}

const FILE_CONFIG_WITH_FEDERATION_NODES = `{
	"federation-nodes": [
		{"Key":"dfc06c5be24a67adee80b35ab4f147bb1a35c55ff85eda69f40ef827bddec173","IP":"192.168.199.2","Port":4400},
		{"Key":"92d469d7c004cc0b24a192d9457836bf38effa27536627ef60718b00b0f33152","IP":"192.168.199.3","Port":4400},
		{"Key":"a899b318e65915aa2de02841eeb72fe51fddad96014b73800ca788a547f8cce0","IP":"192.168.199.4","Port":4400}
	]
}`

func TestSetFederationNodes(t *testing.T) {
	cfg, err := NewEmptyFileConfig(FILE_CONFIG_WITH_FEDERATION_NODES)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, 3, len(cfg.FederationNodes(0)))

	keyPair := keys.Ed25519KeyPairForTests(0)

	node1 := &hardCodedFederationNode{
		nodePublicKey: keyPair.PublicKey(),
	}

	require.EqualValues(t, node1, cfg.FederationNodes(0)[keyPair.PublicKey().KeyForMap()])
}

func TestSetGossipPeers(t *testing.T) {
	cfg, err := NewEmptyFileConfig(FILE_CONFIG_WITH_FEDERATION_NODES)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, 3, len(cfg.GossipPeers(0)))

	keyPair := keys.Ed25519KeyPairForTests(0)

	node1 := &hardCodedGossipPeer{
		gossipEndpoint: "192.168.199.2",
		gossipPort:     4400,
	}

	require.EqualValues(t, node1, cfg.GossipPeers(0)[keyPair.PublicKey().KeyForMap()])
}

const FILE_CONFIG_WITH_GOSSIP_PORT = `{"gossip-port": 4500}`

func TestSetGossipPort(t *testing.T) {
	cfg, err := NewEmptyFileConfig(FILE_CONFIG_WITH_GOSSIP_PORT)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, 4500, cfg.GossipListenPort())
}

func TestMergeWithFileConfig(t *testing.T) {
	nodes := make(map[string]FederationNode)
	peers := make(map[string]GossipPeer)
	keyPair := keys.Ed25519KeyPairForTests(2)

	cfg := ForAcceptanceTests(nodes, peers,
		keyPair.PublicKey(), keyPair.PrivateKey(), keyPair.PublicKey(),
		consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS, 30)

	require.EqualValues(t, keyPair.PublicKey(), cfg.NodePublicKey())
	require.EqualValues(t, 0, len(cfg.FederationNodes(0)))

	cfg.MergeWithFileConfig(FILE_CONFIG_CONTENTS)

	newKeyPair := keys.Ed25519KeyPairForTests(0)

	require.EqualValues(t, 3, len(cfg.FederationNodes(0)))
	require.EqualValues(t, newKeyPair.PublicKey(), cfg.NodePublicKey())
}
