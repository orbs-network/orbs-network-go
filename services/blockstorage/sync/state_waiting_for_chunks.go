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
	}

	return nil
}

func (s *waitingForChunksState) blockCommitted(blockHeight primitives.BlockHeight) {
	panic("implement me")
}

func (s *waitingForChunksState) gotAvailabilityResponse(message *gossipmessages.BlockAvailabilityResponseMessage) {
	panic("implement me")
}

func (s *waitingForChunksState) gotBlocks(source primitives.Ed25519PublicKey, message *gossipmessages.BlockSyncResponseMessage) {
	s.blocks = message
	s.process <- struct{}{}
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
