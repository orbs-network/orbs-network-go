package sync

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestIdleStateStaysIdleOnCommit(t *testing.T) {
	idle := createIdleState(3 * time.Millisecond)
	var next syncState = nil
	// in parallel, we will request to advance to the next state and commit blocks,
	// while blocks are committed we should never advance
	go func() { next = idle.next() }()
	go func() {
		for i := 11; i < 10000; i++ {
			time.Sleep(1 * time.Millisecond)
			idle.blockCommitted(primitives.BlockHeight(i))
		}
	}()
	time.Sleep(10 * time.Millisecond)
	require.Nil(t, next, "next state should not have happened yet")
}

func TestIdleStateMovesToCollectingOnNoCommitTimeout(t *testing.T) {
	idle := createIdleState(3 * time.Millisecond)
	var next syncState = nil
	next = idle.next()
	_, ok := next.(*collectingAvailabilityResponsesState)
	require.True(t, ok, "next state should be collecting availability responses")
}
