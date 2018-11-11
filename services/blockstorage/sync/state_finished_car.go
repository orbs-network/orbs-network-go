package sync

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"math/rand"
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
	start := time.Now()
	defer s.metrics.stateLatency.RecordSince(start) // runtime metric

	if ctx.Err() == context.Canceled { // system is terminating and we do not select on channels in this state
		return nil
	}

	c := len(s.responses)
	if c == 0 {
		s.logger.Info("no responses received")
		s.metrics.timesNoResponses.Inc()
		return s.factory.CreateIdleState()
	}
	s.metrics.timesWithResponses.Inc()
	s.logger.Info("selecting from received sources", log.Int("sources-count", c))
	syncSource := s.responses[rand.Intn(c)]
	syncSourceKey := syncSource.Sender.SenderPublicKey()

	return s.factory.CreateWaitingForChunksState(syncSourceKey)
}

func (s *finishedCARState) blockCommitted(ctx context.Context) {
	return
}

func (s *finishedCARState) gotAvailabilityResponse(ctx context.Context, message *gossipmessages.BlockAvailabilityResponseMessage) {
	return
}

func (s *finishedCARState) gotBlocks(ctx context.Context, message *gossipmessages.BlockSyncResponseMessage) {
	return
}
