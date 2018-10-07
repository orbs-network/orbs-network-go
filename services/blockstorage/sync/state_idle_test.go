package sync

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestIdleStateStaysIdleOnCommit(t *testing.T) {
	ctx := context.Background()
	sf := stateFactory{}
	idle := sf.CreateIdleState(3 * time.Millisecond)
	var next syncState = nil
	latch := make(chan struct{})
	go func() {
		next = idle.processState(ctx)
		latch <- struct{}{}
	}()
	idle.blockCommitted(primitives.BlockHeight(11))
	<-latch
	require.True(t, next != idle, "processState state should be a different idle state (which was restarted)")
}

func TestIdleStateMovesToCollectingOnNoCommitTimeout(t *testing.T) {
	ctx := context.Background()
	sf := stateFactory{}
	idle := sf.CreateIdleState(3 * time.Millisecond)
	next := idle.processState(ctx)
	_, ok := next.(*collectingAvailabilityResponsesState)
	require.True(t, ok, "processState state should be collecting availability responses")
}

func TestIdleStateTerminatesOnContextTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sf := stateFactory{}
	idle := sf.CreateIdleState(3 * time.Millisecond)
	next := idle.processState(ctx)

	require.Nil(t, next, "context termination should return a nil new state")
}
