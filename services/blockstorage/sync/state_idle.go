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
}

func createIdleState(noCommitTimeout time.Duration) *idleState {
	return &idleState{
		noCommitTimeout: noCommitTimeout,
		noCommitTimer:   synchronization.NewTimer(noCommitTimeout),
	}
}

func (s *idleState) next() syncState {
	select {
	case <-s.noCommitTimer.C:
		return &collectingAvailabilityResponsesState{}
	}
}

func (s *idleState) blockCommitted(blockHeight primitives.BlockHeight) {
	s.noCommitTimer.Reset(s.noCommitTimeout)
}

func (s *idleState) gotAvailabilityResponse(message gossipmessages.BlockAvailabilityResponseMessage) {
	return
}

func (s *idleState) gotBlocks(source primitives.Ed25519PublicKey, blocks []*protocol.BlockPairContainer) {
	return
}
