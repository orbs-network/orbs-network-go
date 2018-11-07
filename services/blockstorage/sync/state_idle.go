package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"time"
)

type idleState struct {
	createNoCommitTimeoutTimer func() *synchronization.Timer
	logger                     log.BasicLogger
	factory                    *stateFactory
	conduit                    *blockSyncConduit
	metrics                    idleStateMetrics
}

func (s *idleState) name() string {
	return "idle-state"
}

func (s *idleState) String() string {
	return s.name()
}

func (s *idleState) processState(ctx context.Context) syncState {
	start := time.Now()
	defer s.metrics.stateLatency.RecordSince(start) // runtime metric

	noCommitTimer := s.createNoCommitTimeoutTimer()
	select {
	case <-noCommitTimer.C:
		s.logger.Info("starting sync after no-commit timer expired")
		s.metrics.timesExpired.Inc()
		return s.factory.CreateCollectingAvailabilityResponseState()
	case <-s.conduit.idleReset:
		s.metrics.timesReset.Inc()
		return s.factory.CreateIdleState()
	case <-ctx.Done():
		return nil
	}
}

func (s *idleState) blockCommitted(ctx context.Context) {
	select {
	case s.conduit.idleReset <- struct{}{}:
		s.logger.Info("sync got new block commit")
	case <-ctx.Done():
		s.logger.Info("terminated on writing new block notification", log.String("context-message", ctx.Err().Error()))
	}
}

func (s *idleState) gotAvailabilityResponse(ctx context.Context, message *gossipmessages.BlockAvailabilityResponseMessage) {
	return
}

func (s *idleState) gotBlocks(ctx context.Context, message *gossipmessages.BlockSyncResponseMessage) {
	return
}
