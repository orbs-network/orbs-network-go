package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

type idleState struct {
	config        blockSyncConfig
	noCommitTimer *synchronization.Timer
	restartIdle   chan struct{}
	sf            *stateFactory
}

func (s *idleState) name() string {
	return "idle-state"
}

func (s *idleState) processState(ctx context.Context) syncState {
	select {
	case <-s.noCommitTimer.C:
		return &collectingAvailabilityResponsesState{}
	case <-s.restartIdle:
		return s.sf.CreateIdleState()
	case <-ctx.Done():
		return nil
	}
}

func (s *idleState) blockCommitted(blockHeight primitives.BlockHeight) {
	s.restartIdle <- struct{}{}
}

func (s *idleState) gotAvailabilityResponse(message *gossipmessages.BlockAvailabilityResponseMessage) {
	return
}

func (s *idleState) gotBlocks(message *gossipmessages.BlockSyncResponseMessage) {
	return
}
