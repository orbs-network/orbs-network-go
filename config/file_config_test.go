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
	"active-consensus-algo": 1
}
`

func TestFileConfigConstructor(t *testing.T) {
	cfg, err := NewFileConfig(FILE_CONFIG_CONTENTS)

	require.NotNil(t, cfg)
	require.NoError(t, err)
}

func TestFileConfigSetUint32(t *testing.T) {
	cfg, err := NewFileConfig(FILE_CONFIG_CONTENTS)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, 999, cfg.BlockSyncBatchSize())
}

func TestFileConfigSetDuration(t *testing.T) {
	cfg, err := NewFileConfig(FILE_CONFIG_CONTENTS)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, 10*time.Minute, cfg.BlockSyncCollectResponseTimeout())
}

func TestSetNodePublicKey(t *testing.T) {
	cfg, err := NewFileConfig(FILE_CONFIG_CONTENTS)

	keyPair := keys.Ed25519KeyPairForTests(0)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, keyPair.PublicKey(), cfg.NodePublicKey())
}

func TestSetNodePrivateKey(t *testing.T) {
	cfg, err := NewFileConfig(FILE_CONFIG_CONTENTS)

	keyPair := keys.Ed25519KeyPairForTests(0)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, keyPair.PrivateKey(), cfg.NodePrivateKey())
}

func TestSetConstantConsensusLeader(t *testing.T) {
	cfg, err := NewFileConfig(FILE_CONFIG_CONTENTS)

	keyPair := keys.Ed25519KeyPairForTests(1)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, keyPair.PublicKey(), cfg.ConstantConsensusLeader())
}

func TestSetActiveConsensusAlgo(t *testing.T) {
	cfg, err := NewFileConfig(FILE_CONFIG_CONTENTS)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS, cfg.ActiveConsensusAlgo())
}
