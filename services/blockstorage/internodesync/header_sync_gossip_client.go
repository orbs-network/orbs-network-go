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
)

type headerSyncClient struct {
	gossip      gossiptopics.HeaderSync
	storage     BlockSyncStorage
	logger      log.Logger
	batchSize   func() uint32
	nodeAddress func() primitives.NodeAddress
}

func newHeaderSyncGossipClient(
	g gossiptopics.HeaderSync,
	s BlockSyncStorage,
	l log.Logger,
	batchSize func() uint32,
	na func() primitives.NodeAddress) *headerSyncClient {

	return &headerSyncClient{
		gossip:      g,
		storage:     s,
		logger:      l,
		batchSize:   batchSize,
		nodeAddress: na,
	}
}


func (c *headerSyncClient) petitionerBroadcastHeaderAvailabilityRequest(ctx context.Context) error {
	logger := c.logger.WithTags(trace.LogFieldFrom(ctx))

	out, err := c.storage.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
	if err != nil {
		return err
	}
	lastCommittedBlockHeight := out.LastCommittedBlockHeight

	firstBlockHeight := primitives.BlockHeight(0)
	lastBlockHeight := lastCommittedBlockHeight + 1

	logger.Info("broadcast header availability request",
		log.Uint64("first-block-height", uint64(firstBlockHeight)),
		log.Uint64("last-block-height", uint64(lastBlockHeight)))

	input := &gossiptopics.HeaderAvailabilityRequestInput{
		Message: &gossipmessages.HeaderAvailabilityRequestMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderNodeAddress: c.nodeAddress(),
			}).Build(),
			SignedBatchRange: (&gossipmessages.HeaderSyncRangeBuilder{
				HeaderType:               gossipmessages.HEADER_TYPE_RESULTS_BLOCK_HEADER_WITH_PROOF,
				LastBlockHeight:          lastBlockHeight,
				FirstBlockHeight:         firstBlockHeight,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
			}).Build(),
		},
	}

	_, err = c.gossip.BroadcastHeaderAvailabilityRequest(ctx, input)
	return err
}

func (c *headerSyncClient) petitionerSendHeaderSyncRequest(ctx context.Context, headerType gossipmessages.HeaderType, recipientNodeAddress primitives.NodeAddress) error {
	out, err := c.storage.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
	if err != nil {
		return err
	}
	lastCommittedBlockHeight := out.LastCommittedBlockHeight
	firstBlockHeight := primitives.BlockHeight(0)
	lastBlockHeight := lastCommittedBlockHeight + 1

	c.logger.Info("sending header sync request", log.Stringable("recipient-address", recipientNodeAddress), log.Stringable("first-block", firstBlockHeight), log.Stringable("last-block", lastBlockHeight), log.Stringable("last-committed-block", lastCommittedBlockHeight))

	request := &gossiptopics.HeaderSyncRequestInput{
		RecipientNodeAddress: recipientNodeAddress,
		Message: &gossipmessages.HeaderSyncRequestMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderNodeAddress: c.nodeAddress(),
			}).Build(),
			SignedChunkRange: (&gossipmessages.HeaderSyncRangeBuilder{
				HeaderType:               headerType,
				LastBlockHeight:          lastBlockHeight,
				FirstBlockHeight:         firstBlockHeight,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
			}).Build(),
		},
	}

	_, err = c.gossip.SendHeaderSyncRequest(ctx, request)
	return err
}
