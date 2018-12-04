package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"math"
)

type TamperingStatePersistence interface {
	adapter.StatePersistence
	Dump() string
	WaitUntilCommittedBlockOfHeight(ctx context.Context, height primitives.BlockHeight) error
}

type TestStatePersistence struct {
	*adapter.InMemoryStatePersistence
	blockTrackerForTests *synchronization.BlockTracker
}

func NewTamperingStatePersistence(metric metric.Registry, log log.BasicLogger) (*TestStatePersistence, adapter.BlockHeightReporter) {
	result := &TestStatePersistence{
		InMemoryStatePersistence: adapter.NewInMemoryStatePersistence(metric),
		blockTrackerForTests:     synchronization.NewBlockTracker(log, 0, math.MaxUint16),
	}
	return result, result.blockTrackerForTests
}

func (t *TestStatePersistence) WaitUntilCommittedBlockOfHeight(ctx context.Context, height primitives.BlockHeight) error {
	return t.blockTrackerForTests.WaitForBlock(ctx, height)
}
