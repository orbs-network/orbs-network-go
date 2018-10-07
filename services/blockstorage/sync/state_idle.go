package sync

import (
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"time"
)

type idleState struct {
	noCommitTimeout time.Duration
	noCommitTimer   *synchronization.Timer
	restartIdle     chan struct{}
}

func createIdleState(noCommitTimeout time.Duration) syncState {
	return &idleState{
		noCommitTimeout: noCommitTimeout,
		noCommitTimer:   synchronization.NewTimer(noCommitTimeout),
		restartIdle:     make(chan struct{}),
	}
}

func (s *idleState) name() string {
	return "idle-state"
}

func (s *idleState) next() syncState {
	select {
	case <-s.noCommitTimer.C:
		return &collectingAvailabilityResponsesState{}
	case <-s.restartIdle:
		return createIdleState(s.noCommitTimeout)
	}
}

func (s *idleState) blockCommitted(blockHeight primitives.BlockHeight) {
	s.restartIdle <- struct{}{}
}

func (s *idleState) gotAvailabilityResponse(message gossipmessages.BlockAvailabilityResponseMessage) {
	return
}

func (s *idleState) gotBlocks(source primitives.Ed25519PublicKey, blocks []*protocol.BlockPairContainer) {
	return
}
