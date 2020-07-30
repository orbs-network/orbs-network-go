// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package internodesync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
)

const UNKNOWN_BLOCK_HEIGHT = primitives.BlockHeight(0)

type blockSyncClient struct {
	gossip      gossiptopics.BlockSync
	storage     BlockSyncStorage
	logger      log.Logger
	batchSize   func() uint32
	nodeAddress func() primitives.NodeAddress
}

func newBlockSyncGossipClient(
	g gossiptopics.BlockSync,
	s BlockSyncStorage,
	l log.Logger,
	batchSize func() uint32,
	na func() primitives.NodeAddress,
) *blockSyncClient {

	return &blockSyncClient{
		gossip:      g,
		storage:     s,
		logger:      l,
		batchSize:   batchSize,
		nodeAddress: na,
	}
}

func (c *blockSyncClient) petitionerUpdateConsensusAlgosAboutLastCommittedBlockInLocalPersistence(ctx context.Context) {
	c.storage.UpdateConsensusAlgosAboutLastCommittedBlockInLocalPersistence(ctx)
}

func (c *blockSyncClient) petitionerBroadcastBlockAvailabilityRequest(ctx context.Context, syncBlocksOrder gossipmessages.SyncBlocksOrder) error {
	logger := c.logger.WithTags(trace.LogFieldFrom(ctx))
	syncState := c.storage.GetSyncState()
	from, to, err := getClientSyncRange(syncState, syncBlocksOrder, primitives.BlockHeight(c.batchSize()), c.logger)
	if err != nil {
		return errors.Wrapf(err, "invalid block availability range request: from %d to %d, blocksOrder: %v", from, to, syncBlocksOrder)
	}
	out, err := c.storage.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
	if err != nil {
		return err
	}
	lastCommittedBlockHeight := out.LastCommittedBlockHeight

	logger.Info("broadcast block availability request",
		log.Uint64("first-block-height", uint64(from)),
		log.Uint64("last-block-height", uint64(to)),
		log.Stringable("blocks-order", syncBlocksOrder))

	input := &gossiptopics.BlockAvailabilityRequestInput{
		Message: &gossipmessages.BlockAvailabilityRequestMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderNodeAddress: c.nodeAddress(),
			}).Build(),
			SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				FirstBlockHeight:         from,
				LastBlockHeight:          to,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
				BlocksOrder:              syncBlocksOrder,
			}).Build(),
		},
	}

	_, err = c.gossip.BroadcastBlockAvailabilityRequest(ctx, input)
	return err
}

func (c *blockSyncClient) petitionerSendBlockSyncRequest(ctx context.Context, syncBlocksOrder gossipmessages.SyncBlocksOrder, blockType gossipmessages.BlockType, recipientNodeAddress primitives.NodeAddress) error {
	logger := c.logger.WithTags(trace.LogFieldFrom(ctx))
	syncState := c.storage.GetSyncState()
	from, to, err := getClientSyncRange(syncState, syncBlocksOrder, primitives.BlockHeight(c.batchSize()), c.logger)
	if err != nil {
		return errors.Wrapf(err, "invalid block availability range request: from %d to %d, blocksOrder: %v", from, to, syncBlocksOrder)
	}
	out, err := c.storage.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
	if err != nil {
		return err
	}
	lastCommittedBlockHeight := out.LastCommittedBlockHeight
	logger.Info("sending block sync request",
		log.Stringable("recipient-address", recipientNodeAddress),
		log.Stringable("first-block", from),
		log.Stringable("last-block", to),
		log.Stringable("last-committed-height", lastCommittedBlockHeight))

	request := &gossiptopics.BlockSyncRequestInput{
		RecipientNodeAddress: recipientNodeAddress,
		Message: &gossipmessages.BlockSyncRequestMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderNodeAddress: c.nodeAddress(),
			}).Build(),
			SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                blockType,
				FirstBlockHeight:         from,
				LastBlockHeight:          to,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
				BlocksOrder:              syncBlocksOrder,
			}).Build(),
		},
	}

	_, err = c.gossip.SendBlockSyncRequest(ctx, request)
	return err
}

// inclusive range
func getClientSyncRange(syncState SyncState, syncBlocksOrder gossipmessages.SyncBlocksOrder, batchSize primitives.BlockHeight, logger log.Logger) (from primitives.BlockHeight, to primitives.BlockHeight, err error) {
	topHeight, inOrderHeight, lastSyncedHeight := syncState.GetSyncStateBlockHeights()
	logger.Info("GetClientSyncRange ", log.Uint64("inOrderHeight", uint64(inOrderHeight)), log.Uint64("lastSyncedHeight", uint64(lastSyncedHeight)), log.Uint64("top", uint64(topHeight)))

	if batchSize == 0 {
		err = errors.New("invalid batch size")
		return
	}
	if syncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_ASCENDING {
		from = inOrderHeight + 1
		to = from + batchSize - 1
		if from > to {
			err = errors.New("calculated -descending- range instead of -ascending-")
		}
	} else if syncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING {
		from = UNKNOWN_BLOCK_HEIGHT
		to = inOrderHeight + 1
		if lastSyncedHeight > inOrderHeight+1 {
			from = lastSyncedHeight - 1
			if (lastSyncedHeight > batchSize) && (lastSyncedHeight-batchSize > inOrderHeight+1) {
				to = lastSyncedHeight - batchSize
			}
			if from < to {
				err = errors.New("calculated -ascending- range instead of -descending-")
			}
		}
	}
	return
}
