// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package internodesync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
)

type blockSyncClient struct {
	gossip      gossiptopics.BlockSync
	storage     BlockSyncStorage
	logger      log.BasicLogger
	batchSize   func() uint32
	nodeAddress func() primitives.NodeAddress
}

func newBlockSyncGossipClient(
	g gossiptopics.BlockSync,
	s BlockSyncStorage,
	l log.BasicLogger,
	batchSize func() uint32,
	na func() primitives.NodeAddress) *blockSyncClient {

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

func (c *blockSyncClient) petitionerBroadcastBlockAvailabilityRequest(ctx context.Context) error {
	logger := c.logger.WithTags(trace.LogFieldFrom(ctx))

	out, err := c.storage.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
	if err != nil {
		return err
	}
	lastCommittedBlockHeight := out.LastCommittedBlockHeight

	firstBlockHeight := lastCommittedBlockHeight + 1
	lastBlockHeight := lastCommittedBlockHeight + primitives.BlockHeight(c.batchSize())

	if firstBlockHeight > lastBlockHeight {
		return errors.Errorf("invalid block request: from %d to %d", firstBlockHeight, lastBlockHeight)
	}

	logger.Info("broadcast block availability request",
		log.Uint64("first-block-height", uint64(firstBlockHeight)),
		log.Uint64("last-block-height", uint64(lastBlockHeight)))

	input := &gossiptopics.BlockAvailabilityRequestInput{
		Message: &gossipmessages.BlockAvailabilityRequestMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderNodeAddress: c.nodeAddress(),
			}).Build(),
			SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				LastBlockHeight:          lastBlockHeight,
				FirstBlockHeight:         firstBlockHeight,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
			}).Build(),
		},
	}

	_, err = c.gossip.BroadcastBlockAvailabilityRequest(ctx, input)
	return err
}

func (c *blockSyncClient) petitionerSendBlockSyncRequest(ctx context.Context, blockType gossipmessages.BlockType, recipientNodeAddress primitives.NodeAddress) error {
	out, err := c.storage.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
	if err != nil {
		return err
	}
	lastCommittedBlockHeight := out.LastCommittedBlockHeight
	firstBlockHeight := lastCommittedBlockHeight + 1
	lastBlockHeight := lastCommittedBlockHeight + primitives.BlockHeight(c.batchSize())

	c.logger.Info("sending block sync request", log.Stringable("recipient-address", recipientNodeAddress), log.Stringable("first-block", firstBlockHeight), log.Stringable("last-block", lastBlockHeight), log.Stringable("last-committed-block", lastCommittedBlockHeight))

	request := &gossiptopics.BlockSyncRequestInput{
		RecipientNodeAddress: recipientNodeAddress,
		Message: &gossipmessages.BlockSyncRequestMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderNodeAddress: c.nodeAddress(),
			}).Build(),
			SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                blockType,
				LastBlockHeight:          lastBlockHeight,
				FirstBlockHeight:         firstBlockHeight,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
			}).Build(),
		},
	}

	_, err = c.gossip.SendBlockSyncRequest(ctx, request)
	return err
}
