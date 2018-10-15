package sync

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

type blockSyncGossipClient struct {
	gossip    gossiptopics.BlockSync
	storage   BlockSyncStorage
	logger    log.BasicLogger
	batchSize uint32
	nodeKey   primitives.Ed25519PublicKey
}

func newBlockSyncGossipClient(
	g gossiptopics.BlockSync,
	s BlockSyncStorage,
	l log.BasicLogger,
	batchSize uint32,
	pk primitives.Ed25519PublicKey) *blockSyncGossipClient {

	return &blockSyncGossipClient{
		gossip:    g,
		storage:   s,
		logger:    l,
		batchSize: batchSize,
		nodeKey:   pk,
	}
}

func (c *blockSyncGossipClient) petitionerBroadcastBlockAvailabilityRequest() error {
	lastCommittedBlockHeight := c.storage.LastCommittedBlockHeight()
	firstBlockHeight := lastCommittedBlockHeight + 1
	lastBlockHeight := lastCommittedBlockHeight + primitives.BlockHeight(c.batchSize)

	c.logger.Info("broadcast block availability request",
		log.Stringable("first-block-height", firstBlockHeight),
		log.Stringable("last-block-height", lastBlockHeight))

	input := &gossiptopics.BlockAvailabilityRequestInput{
		Message: &gossipmessages.BlockAvailabilityRequestMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: c.nodeKey,
			}).Build(),
			SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				LastBlockHeight:          lastBlockHeight,
				FirstBlockHeight:         firstBlockHeight,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
			}).Build(),
		},
	}

	_, err := c.gossip.BroadcastBlockAvailabilityRequest(input)
	return err
}

func (c *blockSyncGossipClient) petitionerSendBlockSyncRequest(blockType gossipmessages.BlockType, senderPublicKey primitives.Ed25519PublicKey) error {
	lastCommittedBlockHeight := c.storage.LastCommittedBlockHeight()

	firstBlockHeight := lastCommittedBlockHeight + 1
	lastBlockHeight := lastCommittedBlockHeight + primitives.BlockHeight(c.batchSize)

	request := &gossiptopics.BlockSyncRequestInput{
		RecipientPublicKey: senderPublicKey,
		Message: &gossipmessages.BlockSyncRequestMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: c.nodeKey,
			}).Build(),
			SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                blockType,
				LastBlockHeight:          lastBlockHeight,
				FirstBlockHeight:         firstBlockHeight,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
			}).Build(),
		},
	}

	_, err := c.gossip.SendBlockSyncRequest(request)
	return err
}
