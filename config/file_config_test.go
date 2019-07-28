package config

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewFromMultipleFiles_ForE2E(t *testing.T) {
	cfg, err := NewFromMultipleFiles([]string{"../docker/test/benchmark-config/node1.json"})
	require.NoError(t, err, "failed parsing config file")

	require.EqualValues(t, "a328846cd5b4979d68a8c58a9bdfeee657b34de7", cfg.NodeAddress().String())
}
