package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
)

type blockSyncGossipClient struct {
	gossip    gossiptopics.BlockSync
	storage   BlockSyncStorage
	logger    log.BasicLogger
	batchSize func() uint32
	nodeKey   func() primitives.Ed25519PublicKey
}

func newBlockSyncGossipClient(
	g gossiptopics.BlockSync,
	s BlockSyncStorage,
	l log.BasicLogger,
	batchSize func() uint32,
	pk func() primitives.Ed25519PublicKey) *blockSyncGossipClient {

	return &blockSyncGossipClient{
		gossip:    g,
		storage:   s,
		logger:    l,
		batchSize: batchSize,
		nodeKey:   pk,
	}
}

func (c *blockSyncGossipClient) petitionerUpdateConsensusAlgos(ctx context.Context) {
	c.storage.UpdateConsensusAlgosAboutLatestCommittedBlock(ctx)
}

func (c *blockSyncGossipClient) petitionerBroadcastBlockAvailabilityRequest(ctx context.Context) error {
	out, err := c.storage.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
	if err != nil {
		return err
	}
	_lastCommittedBlockHeight := out.LastCommittedBlockHeight

	firstBlockHeight := _lastCommittedBlockHeight + 1
	lastBlockHeight := _lastCommittedBlockHeight + primitives.BlockHeight(c.batchSize())

	if firstBlockHeight > lastBlockHeight {
		return errors.Errorf("invalid block request: from %d to %d", firstBlockHeight, lastBlockHeight)
	}

	c.logger.Info("broadcast block availability request",
		log.Stringable("first-block-height", firstBlockHeight),
		log.Stringable("last-block-height", lastBlockHeight))

	input := &gossiptopics.BlockAvailabilityRequestInput{
		Message: &gossipmessages.BlockAvailabilityRequestMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: c.nodeKey(),
			}).Build(),
			SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				LastBlockHeight:          lastBlockHeight,
				FirstBlockHeight:         firstBlockHeight,
				LastCommittedBlockHeight: _lastCommittedBlockHeight,
			}).Build(),
		},
	}

	_, err = c.gossip.BroadcastBlockAvailabilityRequest(ctx, input)
	return err
}

func (c *blockSyncGossipClient) petitionerSendBlockSyncRequest(ctx context.Context, blockType gossipmessages.BlockType, senderPublicKey primitives.Ed25519PublicKey) error {
	out, err := c.storage.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
	if err != nil {
		return err
	}
	_lastCommittedBlockHeight := out.LastCommittedBlockHeight

	firstBlockHeight := _lastCommittedBlockHeight + 1
	lastBlockHeight := _lastCommittedBlockHeight + primitives.BlockHeight(c.batchSize())

	request := &gossiptopics.BlockSyncRequestInput{
		RecipientPublicKey: senderPublicKey,
		Message: &gossipmessages.BlockSyncRequestMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: c.nodeKey(),
			}).Build(),
			SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                blockType,
				LastBlockHeight:          lastBlockHeight,
				FirstBlockHeight:         firstBlockHeight,
				LastCommittedBlockHeight: _lastCommittedBlockHeight,
			}).Build(),
		},
	}

	_, err = c.gossip.SendBlockSyncRequest(ctx, request)
	return err
}
