package blockstorage

import (
	"context"
	"errors"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/sync"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

func (s *service) sourceHandleBlockAvailabilityRequest(ctx context.Context, message *gossipmessages.BlockAvailabilityRequestMessage) error {
	logger := s.logger.WithTags(sync.LogTag, trace.LogFieldFrom(ctx))
	// the three error messages below are added because of issue #437
	if message == nil {
		logger.Error("received block availability request with nil message")
		return nil
	}

	if message.Sender == nil {
		logger.Error("received block availability request with nil Sender")
		return nil
	}

	if message.SignedBatchRange == nil {
		logger.Error("received block availability request with nil SignedBatchRange")
		return nil
	}

	logger.Info("received block availability request",
		log.Stringable("petitioner", message.Sender.SenderPublicKey()),
		log.Stringable("requested-first-block", message.SignedBatchRange.FirstBlockHeight()),
		log.Stringable("requested-last-block", message.SignedBatchRange.LastBlockHeight()),
		log.Stringable("requested-last-committed-block", message.SignedBatchRange.LastCommittedBlockHeight()))

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

	logger.Info("sending the response for availability request",
		log.Stringable("petitioner", response.RecipientPublicKey),
		log.Stringable("first-available-block-height", response.Message.SignedBatchRange.FirstBlockHeight()),
		log.Stringable("last-available-block-height", response.Message.SignedBatchRange.LastBlockHeight()),
		log.Stringable("last-committed-available-block-height", response.Message.SignedBatchRange.LastCommittedBlockHeight()),
		log.Stringable("source", response.Message.Sender.SenderPublicKey()),
	)

	_, err := s.gossip.SendBlockAvailabilityResponse(ctx, response)
	return err
}

func (s *service) sourceHandleBlockSyncRequest(ctx context.Context, message *gossipmessages.BlockSyncRequestMessage) error {
	logger := s.logger.WithTags(sync.LogTag, trace.LogFieldFrom(ctx))

	senderPublicKey := message.Sender.SenderPublicKey()
	blockType := message.SignedChunkRange.BlockType()
	firstRequestedBlockHeight := message.SignedChunkRange.FirstBlockHeight()
	lastRequestedBlockHeight := message.SignedChunkRange.LastBlockHeight()
	lastCommittedBlockHeight := s.LastCommittedBlockHeight()

	logger.Info("received block sync request",
		log.Stringable("petitioner", message.Sender.SenderPublicKey()),
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

	logger.Info("sending blocks to another node via block sync",
		log.Stringable("petitioner", senderPublicKey),
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
	_, err := s.gossip.SendBlockSyncResponse(ctx, response)
	return err
}
