package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"time"
)

type idleState struct {
	idleTimeout time.Duration
	logger      log.BasicLogger
	restartIdle chan struct{}
	sf          *stateFactory
}

func (s *idleState) name() string {
	return "idle-state"
}

func (s *idleState) String() string {
	return s.name()
}

func (s *idleState) processState(ctx context.Context) syncState {
	noCommitTimer := synchronization.NewTimer(s.idleTimeout)
	select {
	case <-noCommitTimer.C:
		s.logger.Info("starting sync after no-commit timer expired")
		return s.sf.CreateCollectingAvailabilityResponseState()
	case <-s.restartIdle:
		return s.sf.CreateIdleState()
	case <-ctx.Done():
		return nil
	}
}

func (s *idleState) blockCommitted() {
	// the default below is important as its possible to get a block committed event before the processState got to the select
	// in a highly concurrent scenario where the state is recreated rapidly, this will deadlock the commit flow
	// its better to skip notification to avoid that deadlock
	select {
	case s.restartIdle <- struct{}{}:
		s.logger.Info("sync got new block commit")
	default:
		s.logger.Info("channel was not ready, skipping notification")
	}
}

func (s *idleState) gotAvailabilityResponse(message *gossipmessages.BlockAvailabilityResponseMessage) {
	return
}

func (s *idleState) gotBlocks(message *gossipmessages.BlockSyncResponseMessage) {
	return
}
