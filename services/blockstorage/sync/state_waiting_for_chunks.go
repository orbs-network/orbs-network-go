package sync

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

type waitingForChunksState struct {
	sf           *stateFactory
	sourceKey    primitives.Ed25519PublicKey
	gossipClient *blockSyncGossipClient
	config       blockSyncConfig
	logger       log.BasicLogger
	abort        chan struct{}
	blocksC      chan *gossipmessages.BlockSyncResponseMessage
}

func (s *waitingForChunksState) name() string {
	return "waiting-for-chunks-state"
}

func (s *waitingForChunksState) String() string {
	return fmt.Sprintf("%s-from-source-%s", s.name(), s.sourceKey)
}

func (s *waitingForChunksState) processState(ctx context.Context) syncState {
	err := s.gossipClient.petitionerSendBlockSyncRequest(gossipmessages.BLOCK_TYPE_BLOCK_PAIR, s.sourceKey)
	if err != nil {
		s.logger.Info("could not request block chunk from source", log.Error(err), log.Stringable("source", s.sourceKey))
		return s.sf.CreateIdleState()
	}

	timeout := synchronization.NewTimer(s.config.BlockSyncCollectChunksTimeout())
	select {
	case <-timeout.C:
		s.logger.Info("timed out when waiting for chunks", log.Stringable("source", s.sourceKey))
		return s.sf.CreateIdleState()
	case blocks := <-s.blocksC:
		s.logger.Info("got blocks from sync", log.Stringable("source", s.sourceKey))
		return s.sf.CreateProcessingBlocksState(blocks)
	case <-s.abort:
		return s.sf.CreateIdleState()
	case <-ctx.Done():
		return nil
	}
}

func (s *waitingForChunksState) blockCommitted() {
	return
}

func (s *waitingForChunksState) gotAvailabilityResponse(message *gossipmessages.BlockAvailabilityResponseMessage) {
	return
}

func (s *waitingForChunksState) gotBlocks(message *gossipmessages.BlockSyncResponseMessage) {
	if !message.Sender.SenderPublicKey().Equal(s.sourceKey) {
		s.logger.Info("byzantine message detected, expected source key does not match incoming",
			log.Stringable("source-key", s.sourceKey),
			log.Stringable("message-key", message.Sender.SenderPublicKey()))
		s.abort <- struct{}{}
	} else {
		s.blocksC <- message
	}
}
