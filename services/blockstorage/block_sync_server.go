// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package blockstorage

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/internodesync"
	"github.com/orbs-network/orbs-network-go/services/gossip"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
)

const UNKNOWN_BLOCK_HEIGHT = primitives.BlockHeight(0)

func (s *Service) HandleBlockAvailabilityRequest(ctx context.Context, input *gossiptopics.BlockAvailabilityRequestInput) (*gossiptopics.EmptyOutput, error) {
	err := s.sourceHandleBlockAvailabilityRequest(ctx, input.Message)
	return nil, err
}

func (s *Service) HandleBlockSyncRequest(ctx context.Context, input *gossiptopics.BlockSyncRequestInput) (*gossiptopics.EmptyOutput, error) {
	err := s.sourceHandleBlockSyncRequest(ctx, input.Message)
	return nil, err
}

func (s *Service) sourceHandleBlockAvailabilityRequest(ctx context.Context, message *gossipmessages.BlockAvailabilityRequestMessage) error {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	if s.nodeSync == nil {
		return nil
	}
	logger.Info("received block availability request",
		log.Stringable("availability-request-message", message))

	out, err := s.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
	if err != nil {
		return err
	}
	lastCommittedBlockHeight := out.LastCommittedBlockHeight
	batchSize := primitives.BlockHeight(s.config.BlockSyncNumBlocksInBatch())
	requestFrom := message.SignedBatchRange.FirstBlockHeight()
	requestTo := message.SignedBatchRange.LastBlockHeight()
	requestSyncBlocksOrder := message.SignedBatchRange.BlocksOrder()

	syncState := s.persistence.GetSyncState()
	responseFrom, responseTo, err := getServerSyncRange(syncState, requestFrom, requestTo, requestSyncBlocksOrder, batchSize)
	if err != nil {
		logger.Info("invalid sync range ", log.Error(err))
		return nil
	}

	response := &gossiptopics.BlockAvailabilityResponseInput{
		RecipientNodeAddress: message.Sender.SenderNodeAddress(),
		Message: &gossipmessages.BlockAvailabilityResponseMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderNodeAddress: s.config.NodeAddress(),
			}).Build(),
			SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:        gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				FirstBlockHeight: responseFrom,
				LastBlockHeight:  responseTo,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
				BlocksOrder: requestSyncBlocksOrder,
			}).Build(),
		},
	}

	logger.Info("sending the response for availability request",
		log.Stringable("availability-response-message", response))

	_, err = s.gossip.SendBlockAvailabilityResponse(ctx, response)
	return err

}


func getServerSyncRange(syncState internodesync.SyncState,
	requestFrom primitives.BlockHeight,
	requestTo primitives.BlockHeight,
	requestSyncBlocksOrder gossipmessages.SyncBlocksOrder,
	batchSize primitives.BlockHeight,
) (responseFrom primitives.BlockHeight, responseTo primitives.BlockHeight, err error) {

	topInOrder := syncState.TopInOrder
	lastSynced := syncState.LastSynced
	top := syncState.Top

	responseFrom = requestFrom
	responseTo = requestTo

	if (requestFrom > top) || (topInOrder < requestFrom && requestFrom < lastSynced) { // server does not hold range beginning
		err = fmt.Errorf("server does not hold requested range requested from(%d) - to(%d) where storage sync state is: top(%d), lastSynced(%d), topInOrder(%d)", uint64(requestFrom), uint64(requestTo), uint64(top), uint64(lastSynced), uint64(topInOrder))
		return
	}

	if requestSyncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_ASCENDING || requestSyncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_RESERVED {
		if requestFrom > requestTo {
			err = errors.New("Invalid requested range ascending order with from > to")
			return
		}
		if requestFrom <= topInOrder { // includes topInOrder = lastSynced = top
			responseTo = min(requestFrom+batchSize-1, requestTo, topInOrder)
		} else if lastSynced <= requestFrom && requestFrom <= top {
			responseTo = min(requestFrom+batchSize-1, requestTo, top)
		}
	}

	if requestSyncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING {
		// assert range - either (from=unknown or from > to ) and to > 0
		if (requestTo == 0) || (requestFrom > 0 && requestTo > requestFrom) {
			err = errors.New("Invalid requested range descending order with from < to")
			return
		}
		if requestFrom == UNKNOWN_BLOCK_HEIGHT { // open ended request
			if (requestTo > top) || (topInOrder < requestTo && requestTo < lastSynced) { // server does not hold range beginning
				err = fmt.Errorf("server does not hold requested range requested from(%d) - to(%d) where storage sync state is: top(%d), lastSynced(%d), topInOrder(%d)", uint64(requestFrom), uint64(requestTo), uint64(top), uint64(lastSynced), uint64(topInOrder))
				return
			}
			responseFrom = top
			if requestTo < topInOrder {
				responseFrom = topInOrder
			}
		}
		if (responseFrom >= batchSize) && (responseFrom-batchSize+1 > responseTo) {
			responseTo = responseFrom - batchSize + 1
		}
		if (lastSynced < top) && (responseTo < lastSynced && lastSynced <= responseFrom) {
			responseTo = lastSynced
		}
	}
	return
}

func min(a, b, c primitives.BlockHeight) primitives.BlockHeight {
	result := a
	if b < result {
		result = b
	}
	if c < result {
		result = c
	}
	return result
}

func reverse(arr []*protocol.BlockPairContainer) {
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}
}

func (s *Service) sourceHandleBlockSyncRequest(ctx context.Context, message *gossipmessages.BlockSyncRequestMessage) error {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	if s.nodeSync == nil {
		return nil
	}
	logger.Info("received block sync chunk request",
		log.Stringable("chunk-request-message", message))

	out, err := s.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
	if err != nil {
		return err
	}
	lastCommittedBlockHeight := out.LastCommittedBlockHeight
	batchSize := primitives.BlockHeight(s.config.BlockSyncNumBlocksInBatch())
	requestFrom := message.SignedChunkRange.FirstBlockHeight()
	requestTo := message.SignedChunkRange.LastBlockHeight()
	requestSyncBlocksOrder := message.SignedChunkRange.BlocksOrder()
	senderNodeAddress := message.Sender.SenderNodeAddress()

	syncState := s.persistence.GetSyncState()
	responseFrom, responseTo, err := getServerSyncRange(syncState, requestFrom, requestTo, requestSyncBlocksOrder, batchSize)
	if err != nil {
		return err
	}

	var blocks []*protocol.BlockPairContainer
	if requestSyncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_ASCENDING || requestSyncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_RESERVED {
		if responseFrom > responseTo {
			return fmt.Errorf("Invalid calculated block slice range: from(%s) - to(%s) ", responseFrom.String(), responseTo.String())
		}
		blocks, _, _, err = s.GetBlockSlice(responseFrom, responseTo)
	}
	if requestSyncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING {
		if responseTo > responseFrom {
			return fmt.Errorf("Invalid calculated block slice range: from(%s) - to(%s) ", responseTo.String(), responseFrom.String())
		}
		blocks, _, _, err = s.GetBlockSlice(responseTo, responseFrom)
		if err == nil { // reverse blocks
			reverse(blocks)
		}
	}
	if err != nil {
		return errors.Wrap(err, "block sync failed reading from block persistence")
	}
	chunkSize := uint(len(blocks))
	for {
		if chunkSize > 0 {
			responseTo = blocks[chunkSize-1].TransactionsBlock.Header.BlockHeight()
		} else {
			responseTo = responseFrom-1
		}
		response := &gossiptopics.BlockSyncResponseInput{
			RecipientNodeAddress: senderNodeAddress,
			Message: &gossipmessages.BlockSyncResponseMessage{
				Sender: (&gossipmessages.SenderSignatureBuilder{
					SenderNodeAddress: s.config.NodeAddress(),
				}).Build(),
				SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
					BlockType:        gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
					FirstBlockHeight: responseFrom,
					LastBlockHeight:  responseTo,
					LastCommittedBlockHeight: lastCommittedBlockHeight,
					BlocksOrder: requestSyncBlocksOrder,
				}).Build(),
				BlockPairs: blocks[:chunkSize],
			},
		}

		logger.Info("sending blocks to another node via block sync",
			log.Stringable("chunk-response-message", response))

		_, err = s.gossip.SendBlockSyncResponse(ctx, response)
		if err != nil {
			if !gossip.IsChunkTooBigError(err) { // A non chunk-size related error, return immediately
				return err
			}
			if chunkSize == 0 { // We just tried sending a zero-length chunk and failed, time to give up
				return err
			}
			chunkSize /= 2 // try again with a smaller chunk
			continue
		}

		return nil
	}
}
