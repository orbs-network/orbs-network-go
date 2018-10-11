package blockstorage

import (
	"errors"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

func (s *service) sourceHandleBlockAvailabilityRequest(message *gossipmessages.BlockAvailabilityRequestMessage) error {
	s.logger.Info("received block availability request", log.Stringable("sender", message.Sender))

	lastCommittedBlockHeight := s.LastCommittedBlockHeight()

	if lastCommittedBlockHeight <= message.SignedBatchRange.LastCommittedBlockHeight() {
		return nil
	}

	firstAvailableBlockHeight := primitives.BlockHeight(1)
	blockType := message.SignedBatchRange.BlockType()

	response := &gossiptopics.BlockAvailabilityResponseInput{
		RecipientPublicKey: message.Sender.SenderPublicKey(),
		Message: &gossipmessages.BlockAvailabilityResponseMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: s.config.NodePublicKey(),
			}).Build(),
			SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                blockType,
				LastBlockHeight:          lastCommittedBlockHeight,
				FirstBlockHeight:         firstAvailableBlockHeight,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
			}).Build(),
		},
	}
	_, err := s.gossip.SendBlockAvailabilityResponse(response)
	return err
}

func (s *service) sourceHandleBlockSyncRequest(message *gossipmessages.BlockSyncRequestMessage) error {
	senderPublicKey := message.Sender.SenderPublicKey()
	blockType := message.SignedChunkRange.BlockType()
	firstRequestedBlockHeight := message.SignedChunkRange.FirstBlockHeight()
	lastRequestedBlockHeight := message.SignedChunkRange.LastBlockHeight()
	lastCommittedBlockHeight := s.LastCommittedBlockHeight()

	s.logger.Info("received block sync request",
		log.Stringable("sender", message.Sender),
		log.Stringable("first-requested-block-height", firstRequestedBlockHeight),
		log.Stringable("last-requested-block-height", lastRequestedBlockHeight),
		log.Stringable("last-committed-block-height", lastCommittedBlockHeight))

	if lastCommittedBlockHeight <= firstRequestedBlockHeight {
		return errors.New("firstBlockHeight is greater or equal to lastCommittedBlockHeight")
	}

	if firstRequestedBlockHeight-lastCommittedBlockHeight > primitives.BlockHeight(s.config.BlockSyncBatchSize()-1) {
		lastRequestedBlockHeight = firstRequestedBlockHeight + primitives.BlockHeight(s.config.BlockSyncBatchSize()-1)
	}

	blocks, firstAvailableBlockHeight, lastAvailableBlockHeight := s.GetBlocks(firstRequestedBlockHeight, lastRequestedBlockHeight)

	s.logger.Info("sending blocks to another node via block sync",
		log.Stringable("recipient", senderPublicKey),
		log.Stringable("first-available-block-height", firstAvailableBlockHeight),
		log.Stringable("last-available-block-height", lastAvailableBlockHeight))

	response := &gossiptopics.BlockSyncResponseInput{
		RecipientPublicKey: senderPublicKey,
		Message: &gossipmessages.BlockSyncResponseMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: s.config.NodePublicKey(),
			}).Build(),
			SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                blockType,
				FirstBlockHeight:         firstAvailableBlockHeight,
				LastBlockHeight:          lastAvailableBlockHeight,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
			}).Build(),
			BlockPairs: blocks,
		},
	}
	_, err := s.gossip.SendBlockSyncResponse(response)
	return err
}
