package config

import (
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestFileConfigConstructor(t *testing.T) {
	cfg, err := newEmptyFileConfig(`{}`)

	require.NotNil(t, cfg)
	require.NoError(t, err)
}

func TestFileConfigSetUint32(t *testing.T) {
	cfg, err := newEmptyFileConfig(`{"block-sync-batch-size": 999}`)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, 999, cfg.BlockSyncBatchSize())
}

func TestFileConfigSetDuration(t *testing.T) {
	cfg, err := newEmptyFileConfig(`{"block-sync-collect-response-timeout": "10m"}`)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, 10*time.Minute, cfg.BlockSyncCollectResponseTimeout())
}

func TestSetNodePublicKey(t *testing.T) {
	cfg, err := newEmptyFileConfig(`{"node-public-key": "dfc06c5be24a67adee80b35ab4f147bb1a35c55ff85eda69f40ef827bddec173"}`)

	keyPair := keys.Ed25519KeyPairForTests(0)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, keyPair.PublicKey(), cfg.NodePublicKey())
}

func TestSetNodePrivateKey(t *testing.T) {
	cfg, err := newEmptyFileConfig(`{"node-private-key": "93e919986a22477fda016789cca30cb841a135650938714f85f0000a65076bd4dfc06c5be24a67adee80b35ab4f147bb1a35c55ff85eda69f40ef827bddec173"}`)

	keyPair := keys.Ed25519KeyPairForTests(0)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, keyPair.PrivateKey(), cfg.NodePrivateKey())
}

func TestSetConstantConsensusLeader(t *testing.T) {
	cfg, err := newEmptyFileConfig(`{"constant-consensus-leader": "92d469d7c004cc0b24a192d9457836bf38effa27536627ef60718b00b0f33152"}`)

	keyPair := keys.Ed25519KeyPairForTests(1)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, keyPair.PublicKey(), cfg.ConstantConsensusLeader())
}

func TestSetActiveConsensusAlgo(t *testing.T) {
	cfg, err := newEmptyFileConfig(`{"active-consensus-algo": 999}`)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, 999, cfg.ActiveConsensusAlgo())
}

func TestSetFederationNodes(t *testing.T) {
	cfg, err := newEmptyFileConfig(`{
	"federation-nodes": [
		{"Key":"dfc06c5be24a67adee80b35ab4f147bb1a35c55ff85eda69f40ef827bddec173","IP":"192.168.199.2","Port":4400},
		{"Key":"92d469d7c004cc0b24a192d9457836bf38effa27536627ef60718b00b0f33152","IP":"192.168.199.3","Port":4400},
		{"Key":"a899b318e65915aa2de02841eeb72fe51fddad96014b73800ca788a547f8cce0","IP":"192.168.199.4","Port":4400}
	]
}`)

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
	cfg, err := newEmptyFileConfig(`{
	"federation-nodes": [
		{"Key":"dfc06c5be24a67adee80b35ab4f147bb1a35c55ff85eda69f40ef827bddec173","IP":"192.168.199.2","Port":4400},
		{"Key":"92d469d7c004cc0b24a192d9457836bf38effa27536627ef60718b00b0f33152","IP":"192.168.199.3","Port":4400},
		{"Key":"a899b318e65915aa2de02841eeb72fe51fddad96014b73800ca788a547f8cce0","IP":"192.168.199.4","Port":4400}
	]
}`)

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

func TestSetGossipPort(t *testing.T) {
	cfg, err := newEmptyFileConfig(`{"gossip-port": 4500}`)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, 4500, cfg.GossipListenPort())
}

func TestMergeWithFileConfig(t *testing.T) {
	nodes := make(map[string]FederationNode)
	keyPair := keys.Ed25519KeyPairForTests(2)

	cfg := ForAcceptanceTestNetwork(nodes,
		keyPair.PublicKey(),
		consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS, 30, 100)

	require.EqualValues(t, 0, len(cfg.FederationNodes(0)))

	cfg.MergeWithFileConfig(`
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
`)

	newKeyPair := keys.Ed25519KeyPairForTests(0)

	require.EqualValues(t, 3, len(cfg.FederationNodes(0)))
	require.EqualValues(t, newKeyPair.PublicKey(), cfg.NodePublicKey())
}
