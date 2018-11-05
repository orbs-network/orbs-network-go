package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"time"
)

type collectingAvailabilityResponsesState struct {
	sf             *stateFactory
	gossipClient   *blockSyncGossipClient
	collectTimeout func() time.Duration
	logger         log.BasicLogger
	conduit        *blockSyncConduit
	m              collectingStateMetrics
}

func (s *collectingAvailabilityResponsesState) name() string {
	return "collecting-availability-responses"
}

func (s *collectingAvailabilityResponsesState) String() string {
	return s.name()
}

func (s *collectingAvailabilityResponsesState) processState(ctx context.Context) syncState {
	start := time.Now()
	defer s.m.stateLatency.RecordSince(start) // runtime metric

	responses := []*gossipmessages.BlockAvailabilityResponseMessage{}

	s.gossipClient.petitionerUpdateConsensusAlgos(ctx)
	err := s.gossipClient.petitionerBroadcastBlockAvailabilityRequest(ctx)
	if err != nil {
		s.logger.Info("failed to broadcast block availability request", log.Error(err))
		return s.sf.CreateIdleState()
	}

	waitForResponses := synchronization.NewTimer(s.collectTimeout())
	for { // the forever is because of responses handling loop
		select {
		case <-waitForResponses.C:
			s.m.timesSuccessful.Inc()
			s.logger.Info("finished waiting for responses", log.Int("responses-received", len(responses)))
			return s.sf.CreateFinishedCARState(responses)
		case r := <-s.conduit.responses:
			responses = append(responses, r)
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *collectingAvailabilityResponsesState) blockCommitted(ctx context.Context) {
	return
}

func (s *collectingAvailabilityResponsesState) gotAvailabilityResponse(ctx context.Context, message *gossipmessages.BlockAvailabilityResponseMessage) {
	s.logger.Info("got a new availability response", log.Stringable("response-source", message.Sender.SenderPublicKey()))
	select {
	case s.conduit.responses <- message:
	case <-ctx.Done():
		s.logger.Info("terminated on writing new availability response",
			log.String("context-message", ctx.Err().Error()),
			log.Stringable("response-source", message.Sender.SenderPublicKey()))
	}
}

func (s *collectingAvailabilityResponsesState) gotBlocks(ctx context.Context, message *gossipmessages.BlockSyncResponseMessage) {
	s.logger.Info("got a block chunk in availability response state", log.Stringable("block-source", message.Sender.SenderPublicKey()))
	return
}
