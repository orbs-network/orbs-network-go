package config

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestValidateConfig(t *testing.T) {
	require.NotPanics(t, func() {
		Validate(defaultProductionConfig())
	})
}

func TestValidateConfig_PanicsOnInvalidValue(t *testing.T) {
	cfg := defaultProductionConfig()
	cfg.SetDuration(BLOCK_SYNC_NO_COMMIT_INTERVAL, 1*time.Millisecond)

	require.Panics(t, func() {
		Validate(cfg)
	})
}
