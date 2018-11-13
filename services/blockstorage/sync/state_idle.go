package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"time"
)

type idleState struct {
	createTimer func() *synchronization.Timer
	logger      log.BasicLogger
	factory     *stateFactory
	conduit     *blockSyncConduit
	metrics     idleStateMetrics
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
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	noCommitTimer := s.createTimer()
	select {
	case <-noCommitTimer.C:
		logger.Info("starting sync after no-commit timer expired")
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
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	select {
	case s.conduit.idleReset <- struct{}{}:
		logger.Info("sync got new block commit")
	case <-ctx.Done():
		logger.Info("terminated on writing new block notification", log.String("context-message", ctx.Err().Error()))
	}
}

func (s *idleState) gotAvailabilityResponse(ctx context.Context, message *gossipmessages.BlockAvailabilityResponseMessage) {
	return
}

func (s *idleState) gotBlocks(ctx context.Context, message *gossipmessages.BlockSyncResponseMessage) {
	return
}
