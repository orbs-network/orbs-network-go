package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"time"
)

type idleState struct {
	idleTimeout func() time.Duration
	logger      log.BasicLogger
	sf          *stateFactory
	conduit     *blockSyncConduit
	latency     *metric.Histogram
}

func (s *idleState) name() string {
	return "idle-state"
}

func (s *idleState) String() string {
	return s.name()
}

func (s *idleState) processState(ctx context.Context) syncState {
	start := time.Now()
	defer s.latency.RecordSince(start) // runtime metric

	noCommitTimer := synchronization.NewTimer(s.idleTimeout())
	select {
	case <-noCommitTimer.C:
		s.logger.Info("starting sync after no-commit timer expired")
		return s.sf.CreateCollectingAvailabilityResponseState()
	case <-s.conduit.idleReset:
		return s.sf.CreateIdleState()
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
