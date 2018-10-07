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
	go func() { next = idle.next() }()
	idle.blockCommitted(primitives.BlockHeight(11))
	require.True(t, &next != &idle, "next state should be a different idle state (which was restarted)")
}

func TestIdleStateMovesToCollectingOnNoCommitTimeout(t *testing.T) {
	idle := createIdleState(3 * time.Millisecond)
	var next syncState = nil
	next = idle.next()
	_, ok := next.(*collectingAvailabilityResponsesState)
	require.True(t, ok, "next state should be collecting availability responses")
}
