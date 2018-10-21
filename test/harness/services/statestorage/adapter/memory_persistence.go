package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"time"
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

func NewTamperingStatePersistence() TamperingStatePersistence {
	return &TestStatePersistence{
		InMemoryStatePersistence: adapter.NewInMemoryStatePersistence(),
		blockTrackerForTests:     synchronization.NewBlockTracker(0, 64000, time.Duration(10*time.Second)),
	}
}

func (t *TestStatePersistence) WriteState(height primitives.BlockHeight, contractStateDiffs []*protocol.ContractStateDiff) error {
	err := t.InMemoryStatePersistence.WriteState(height, contractStateDiffs)
	if err != nil {
		return err
	}

	t.blockTrackerForTests.IncrementHeight()
	return nil
}

func (t *TestStatePersistence) WaitUntilCommittedBlockOfHeight(ctx context.Context, height primitives.BlockHeight) error {
	return t.blockTrackerForTests.WaitForBlock(ctx, height)
}
