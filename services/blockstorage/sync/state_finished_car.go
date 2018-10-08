package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
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
	c := len(s.responses)
	if c == 0 {
		return s.sf.CreateIdleState()
	}
	s.logger.Info("selecting from received sources", log.Int("sources-count", c))
	syncSource := s.responses[rand.Intn(c)]
	syncSourceKey := syncSource.Sender.SenderPublicKey()

	return s.sf.CreateWaitingForChunksState(syncSourceKey)
}

func (s *finishedCARState) blockCommitted(blockHeight primitives.BlockHeight) {
	return
}

func (s *finishedCARState) gotAvailabilityResponse(message *gossipmessages.BlockAvailabilityResponseMessage) {
	return
}

func (s *finishedCARState) gotBlocks(message *gossipmessages.BlockSyncResponseMessage) {
	return
}
