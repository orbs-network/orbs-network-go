package sync

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"time"
)

type waitingForChunksState struct {
	factory                         *stateFactory
	sourceKey                       primitives.Ed25519PublicKey
	gossipClient                    *blockSyncGossipClient
	createWaitForChunksTimeoutTimer func() *synchronization.Timer
	logger                          log.BasicLogger
	abort                           chan struct{}
	conduit                         *blockSyncConduit
	metrics                         waitingStateMetrics
}

func (s *waitingForChunksState) name() string {
	return "waiting-for-chunks-state"
}

func (s *waitingForChunksState) String() string {
	return fmt.Sprintf("%s-from-source-%s", s.name(), s.sourceKey)
}

func (s *waitingForChunksState) processState(ctx context.Context) syncState {
	start := time.Now()
	defer s.metrics.stateLatency.RecordSince(start) // runtime metric

	err := s.gossipClient.petitionerSendBlockSyncRequest(ctx, gossipmessages.BLOCK_TYPE_BLOCK_PAIR, s.sourceKey)
	if err != nil {
		s.logger.Info("could not request block chunk from source", log.Error(err), log.Stringable("source", s.sourceKey))
		return s.factory.CreateIdleState()
	}

	timeout := s.createWaitForChunksTimeoutTimer()
	select {
	case <-timeout.C:
		s.logger.Info("timed out when waiting for chunks", log.Stringable("source", s.sourceKey))
		s.metrics.timesTimeout.Inc()
		return s.factory.CreateIdleState()
	case blocks := <-s.conduit.blocks:
		s.logger.Info("got blocks from sync", log.Stringable("source", s.sourceKey))
		s.metrics.timesSuccessful.Inc()
		return s.factory.CreateProcessingBlocksState(blocks)
	case <-s.abort:
		s.metrics.timesByzantine.Inc()
		return s.factory.CreateIdleState()
	case <-ctx.Done():
		return nil
	}
}

func (s *waitingForChunksState) blockCommitted(ctx context.Context) {
	return
}

func (s *waitingForChunksState) gotAvailabilityResponse(ctx context.Context, message *gossipmessages.BlockAvailabilityResponseMessage) {
	return
}

func (s *waitingForChunksState) gotBlocks(ctx context.Context, message *gossipmessages.BlockSyncResponseMessage) {
	if !message.Sender.SenderPublicKey().Equal(s.sourceKey) {
		s.logger.Info("byzantine message detected, expected source key does not match incoming",
			log.Stringable("source-key", s.sourceKey),
			log.Stringable("message-sender-key", message.Sender.SenderPublicKey()))
		s.abort <- struct{}{}
	} else {
		select {
		case s.conduit.blocks <- message:
		case <-ctx.Done():
			s.logger.Info("terminated on writing new block chunk message",
				log.String("context-message", ctx.Err().Error()),
				log.Stringable("message-sender", message.Sender.SenderPublicKey()))
		}
	}
}
