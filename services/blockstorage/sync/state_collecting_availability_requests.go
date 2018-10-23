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
	responsesC     chan *gossipmessages.BlockAvailabilityResponseMessage
}

func (s *collectingAvailabilityResponsesState) name() string {
	return "collecting-availability-responses"
}

func (s *collectingAvailabilityResponsesState) String() string {
	return s.name()
}

func (s *collectingAvailabilityResponsesState) processState(ctx context.Context) syncState {
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
			s.logger.Info("finished waiting for responses", log.Int("responses-received", len(responses)))
			return s.sf.CreateFinishedCARState(responses)
		case r := <-s.responsesC:
			responses = append(responses, r)
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *collectingAvailabilityResponsesState) blockCommitted() {
	return
}

func (s *collectingAvailabilityResponsesState) gotAvailabilityResponse(message *gossipmessages.BlockAvailabilityResponseMessage) {
	s.logger.Info("got a new availability response", log.Stringable("response-source", message.Sender.SenderPublicKey()))
	select {
	case s.responsesC <- message:
	default:
		s.logger.Info("response channel was not ready, dropping response", log.Stringable("response-source", message.Sender.SenderPublicKey()))
	}
}

func (s *collectingAvailabilityResponsesState) gotBlocks(message *gossipmessages.BlockSyncResponseMessage) {
	return
}
