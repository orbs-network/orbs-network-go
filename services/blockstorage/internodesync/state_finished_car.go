package internodesync

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"time"
)

type finishedCARState struct {
	responses []*gossipmessages.BlockAvailabilityResponseMessage
	logger    log.BasicLogger
	factory   *stateFactory
	metrics   finishedCollectingStateMetrics
}

func (s *finishedCARState) name() string {
	return "finished-collecting-availability-requests-state"
}

func (s *finishedCARState) String() string {
	return fmt.Sprintf("%s-with-%d-responses", s.name(), len(s.responses))
}

func (s *finishedCARState) processState(ctx context.Context) syncState {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	start := time.Now()
	defer s.metrics.stateLatency.RecordSince(start) // runtime metric

	c := len(s.responses)
	if c == 0 {
		logger.Info("no responses received")
		s.metrics.timesNoResponses.Inc()
		return s.factory.CreateIdleState()
	}
	s.metrics.timesWithResponses.Inc()
	logger.Info("selecting from received sources", log.Int("sources-count", c))
	syncSource := s.responses[0] //TODO V1 how do we pick the source?
	syncSourceNodeAddress := syncSource.Sender.SenderNodeAddress()

	if !s.heartbeat(ctx) {
		return nil
	}
	return s.factory.CreateWaitingForChunksState(syncSourceNodeAddress)
}

func (s *finishedCARState) heartbeat(ctx context.Context) bool { // drain conduit and check for shutdown
	for {
		select {
		case <-s.factory.conduit: // nop
		case <-ctx.Done():
			return false // indicate a shutdown was signaled
		default:
			return true
		}
	}
}
