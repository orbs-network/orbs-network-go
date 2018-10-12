package sync

import (
	"context"
	"fmt"
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

func (s *finishedCARState) String() string {
	return fmt.Sprintf("%s-with-%d-responses", s.name(), len(s.responses))
}

func (s *finishedCARState) processState(ctx context.Context) syncState {
	if ctx.Err() == context.Canceled { // system is terminating and we do not select on channels in this state
		return nil
	}

	m := s.logger.Meter("block-sync-finished-car-state")
	defer m.Done()

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
