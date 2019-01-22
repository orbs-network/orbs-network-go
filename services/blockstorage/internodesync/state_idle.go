package internodesync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/synchronization"
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
	for {
		select {
		case e := <-s.conduit.events:
			switch e.(type) {
			case idleResetMessage:
				s.metrics.timesReset.Inc()
				return s.factory.CreateIdleState()
			}
		case <-noCommitTimer.C:
			logger.Info("starting sync after no-commit timer expired")
			s.metrics.timesExpired.Inc()
			return s.factory.CreateCollectingAvailabilityResponseState()
		case <-ctx.Done():
			return nil
		}
	}
}
