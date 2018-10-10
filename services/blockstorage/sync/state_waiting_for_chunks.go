package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

type waitingForChunksState struct {
	sf        *stateFactory
	sourceKey primitives.Ed25519PublicKey
	storage   BlockSyncStorage
	config    blockSyncConfig
	gossip    gossiptopics.BlockSync
	logger    log.BasicLogger
	process   chan struct{}
	abort     chan struct{}
	blocks    *gossipmessages.BlockSyncResponseMessage
}

func (s *waitingForChunksState) name() string {
	return "waiting-for-chunks-state"
}

func (s *waitingForChunksState) processState(ctx context.Context) syncState {
	err := s.petitionerSendBlockSyncRequest(gossipmessages.BLOCK_TYPE_BLOCK_PAIR, s.sourceKey)
	if err != nil {
		s.logger.Info("could not request block chunk from source", log.Error(err), log.Stringable("source", s.sourceKey))
		return s.sf.CreateIdleState()
	}

	timeout := synchronization.NewTimer(s.config.BlockSyncCollectChunksTimeout())
	select {
	case <-timeout.C:
		s.logger.Info("timed out when waiting for chunks", log.Stringable("source", s.sourceKey))
		return s.sf.CreateIdleState()
	case <-s.process:
		s.logger.Info("got blocks from sync", log.Stringable("source", s.sourceKey))
		return s.sf.CreateProcessingBlocksState(s.blocks)
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
		s.blocks = message
		s.process <- struct{}{}
	}
}

func (s *waitingForChunksState) petitionerSendBlockSyncRequest(blockType gossipmessages.BlockType, senderPublicKey primitives.Ed25519PublicKey) error {
	lastCommittedBlockHeight := s.storage.LastCommittedBlockHeight()

	firstBlockHeight := lastCommittedBlockHeight + 1
	lastBlockHeight := lastCommittedBlockHeight + primitives.BlockHeight(s.config.BlockSyncBatchSize())

	request := &gossiptopics.BlockSyncRequestInput{
		RecipientPublicKey: senderPublicKey,
		Message: &gossipmessages.BlockSyncRequestMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: s.config.NodePublicKey(),
			}).Build(),
			SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                blockType,
				LastBlockHeight:          lastBlockHeight,
				FirstBlockHeight:         firstBlockHeight,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
			}).Build(),
		},
	}

	_, err := s.gossip.SendBlockSyncRequest(request)
	return err
}
