package config

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestValidateConfig(t *testing.T) {
	v := validator{log.DefaultTestingLogger(t)}
	require.NotPanics(t, func() {
		v.Validate(defaultProductionConfig())
	})
}

func TestValidateConfig_PanicsOnInvalidValue(t *testing.T) {
	v := validator{log.DefaultTestingLogger(t)}

	cfg := defaultProductionConfig()
	cfg.SetDuration(BLOCK_SYNC_NO_COMMIT_INTERVAL, 1*time.Millisecond)

	require.Panics(t, func() {
		v.Validate(cfg)
	})
}
