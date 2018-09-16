package config

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const FILE_CONFIG_CONTENTS = `
{
	"block-sync-batch-size": 999,
	"block-sync-collect-response-timeout": "10m"
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
