package config

import (
	"context"
	"github.com/orbs-network/scribe/log"
	"time"
)

type ChangeHandler func(newConfig *MapBasedConfig)

type Loader interface {
	Load() (*MapBasedConfig, error)
	OnConfigChanged(handler ChangeHandler)
	ListenForChanges(ctx context.Context, logger log.Logger, pollInterval time.Duration, onShutdown func())
}
