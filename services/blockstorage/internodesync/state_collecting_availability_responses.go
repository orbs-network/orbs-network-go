package internodesync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"time"
)

type collectingAvailabilityResponsesState struct {
	factory      *stateFactory
	gossipClient *blockSyncGossipClient
	createTimer  func() *synchronization.Timer
	logger       log.BasicLogger
	conduit      chan interface{}
	metrics      collectingStateMetrics
}

func (s *collectingAvailabilityResponsesState) name() string {
	return "collecting-availability-responses"
}

func (s *collectingAvailabilityResponsesState) String() string {
	return s.name()
}

func (s *collectingAvailabilityResponsesState) processState(ctx context.Context) syncState {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	start := time.Now()
	defer s.metrics.stateLatency.RecordSince(start) // runtime metric

	var responses []*gossipmessages.BlockAvailabilityResponseMessage

	s.gossipClient.petitionerUpdateConsensusAlgos(ctx)
	err := s.gossipClient.petitionerBroadcastBlockAvailabilityRequest(ctx)
	if err != nil {
		logger.Info("failed to broadcast block availability request", log.Error(err))
		return s.factory.CreateIdleState()
	}

	waitForResponses := s.createTimer()
	for {
		select {
		case <-waitForResponses.C:
			s.metrics.timesSuccessful.Inc()
			logger.Info("finished waiting for responses", log.Int("responses-received", len(responses)))
			return s.factory.CreateFinishedCARState(responses)
		case e := <-s.conduit:
			switch r := e.(type) {
			case *gossipmessages.BlockAvailabilityResponseMessage:
				responses = append(responses, r)
				logger.Info("got a new availability response", log.Stringable("response-source", r.Sender.SenderNodeAddress()))
			}
		case <-ctx.Done():
			return nil
		}
	}
}
