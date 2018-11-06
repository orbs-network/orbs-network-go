package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
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
		blockTrackerForTests:     synchronization.NewBlockTracker(0, 64000),
	}
}

func (t *TestStatePersistence) Write(height primitives.BlockHeight, ts primitives.TimestampNano, root primitives.MerkleSha256, diff adapter.ChainState) error {
	err := t.InMemoryStatePersistence.Write(height, ts, root, diff)
	if err != nil {
		return err
	}

	t.blockTrackerForTests.IncrementHeight()
	return nil
}

func (t *TestStatePersistence) WaitUntilCommittedBlockOfHeight(ctx context.Context, height primitives.BlockHeight) error {
	return t.blockTrackerForTests.WaitForBlock(ctx, height)
}
