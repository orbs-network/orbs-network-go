package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"math/rand"
)

type finishedCARState struct {
	responses []*gossipmessages.BlockAvailabilityResponseMessage
	logger    log.BasicLogger
	sf        *stateFactory
}

func (s *finishedCARState) name() string {
	return "finished-collecting-availability-requests-state"
}

func (s *finishedCARState) processState(ctx context.Context) syncState {
	if ctx.Err() == context.Canceled {
		return nil
	}

	c := len(s.responses)
	if c == 0 {
		s.logger.Info("no responses received")
		return s.sf.CreateIdleState()
	}
	s.logger.Info("selecting from received sources", log.Int("sources-count", c))
	syncSource := s.responses[rand.Intn(c)]
	syncSourceKey := syncSource.Sender.SenderPublicKey()

	return s.sf.CreateWaitingForChunksState(syncSourceKey)
}

func (s *finishedCARState) blockCommitted() {
	return
}

func (s *finishedCARState) gotAvailabilityResponse(message *gossipmessages.BlockAvailabilityResponseMessage) {
	return
}

func (s *finishedCARState) gotBlocks(message *gossipmessages.BlockSyncResponseMessage) {
	return
}
