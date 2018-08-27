package blockstorage

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
	"time"
)

type blockSyncState int

const (
	BLOCK_SYNC_STATE_IDLE                   blockSyncState = 0
	BLOCK_SYNC_STATE_START_SYNC             blockSyncState = 1
	BLOCK_SYNC_PETITIONER_WAITING_FOR_CHUNK blockSyncState = 2
)

type BlockSyncStorage interface {
	GetBlocks(first primitives.BlockHeight, last primitives.BlockHeight) (blocks []*protocol.BlockPairContainer, firstAvailableBlockHeight primitives.BlockHeight, lastAvailableBlockHeight primitives.BlockHeight)
	LastCommittedBlockHeight() primitives.BlockHeight
	CommitBlock(input *services.CommitBlockInput) (*services.CommitBlockOutput, error)
}

type BlockSync struct {
	reporting log.BasicLogger

	config  Config
	storage BlockSyncStorage
	gossip  gossiptopics.BlockSync
	Events  chan interface{}
}

func NewBlockSync(ctx context.Context, storage BlockSyncStorage, gossip gossiptopics.BlockSync, config Config, reporting log.BasicLogger) *BlockSync {
	blockSync := &BlockSync{
		reporting: reporting.For(log.String("flow", "block-sync")),
		storage:   storage,
		gossip:    gossip,
		config:    config,
		Events:    make(chan interface{}),
	}

	go blockSync.mainLoop(ctx)

	return blockSync
}

func (b *BlockSync) mainLoop(ctx context.Context) {
	state := BLOCK_SYNC_STATE_START_SYNC
	var event interface{}
	//var syncSource primitives.Ed25519PublicKey

	syncTrigger := time.AfterFunc(b.config.BlockSyncInterval(), func() {
		state = BLOCK_SYNC_STATE_START_SYNC
		b.reporting.Error("transitioning to", log.Int("state", int(state)))
		b.Events <- nil
	})

	for {
		state = b.dispatchEvent(state, event)

		if state == BLOCK_SYNC_STATE_START_SYNC {
			syncTrigger.Stop()
			syncTrigger.Reset(b.config.BlockSyncInterval())
			b.PetitionerBroadcastBlockAvailabilityRequest()
		}

		select {
		case <-ctx.Done():
			return
		case event = <-b.Events:
			continue
		}
	}
}

func (b *BlockSync) dispatchEvent(state blockSyncState, event interface{}) blockSyncState {
	switch event.(type) {
	case *gossipmessages.BlockAvailabilityRequestMessage:
		message := event.(*gossipmessages.BlockAvailabilityRequestMessage)
		b.SourceHandleBlockAvailabilityRequest(message)
	case *gossipmessages.BlockAvailabilityResponseMessage:
		message := event.(*gossipmessages.BlockAvailabilityResponseMessage)
		err := b.PetitionerHandleBlockAvailabilityResponse(message)

		if err != nil {
			b.reporting.Info("Received bad block availability response", log.Error(err))
		} else {
			return BLOCK_SYNC_PETITIONER_WAITING_FOR_CHUNK
		}
	case *gossipmessages.BlockSyncRequestMessage:
		message := event.(*gossipmessages.BlockSyncRequestMessage)
		b.SourceHandleBlockSyncRequest(message)
		return BLOCK_SYNC_STATE_IDLE
	case *gossipmessages.BlockSyncResponseMessage:
		message := event.(*gossipmessages.BlockSyncResponseMessage)
		b.PetitionerHandleBlockSyncResponse(message)
		return BLOCK_SYNC_STATE_IDLE
	}

	return state
}

func (b *BlockSync) PetitionerBroadcastBlockAvailabilityRequest() {
	lastCommittedBlockHeight := b.storage.LastCommittedBlockHeight()
	firstBlockHeight := lastCommittedBlockHeight + 1
	lastBlockHeight := b.storage.LastCommittedBlockHeight() + primitives.BlockHeight(b.config.BlockSyncBatchSize())

	input := &gossiptopics.BlockAvailabilityRequestInput{
		Message: &gossipmessages.BlockAvailabilityRequestMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: b.config.NodePublicKey(),
			}).Build(),
			SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				LastBlockHeight:          lastBlockHeight,
				FirstBlockHeight:         firstBlockHeight,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
			}).Build(),
		},
	}

	b.gossip.BroadcastBlockAvailabilityRequest(input)
}

func (b *BlockSync) PetitionerHandleBlockSyncResponse(message *gossipmessages.BlockSyncResponseMessage) {
	firstAvailableBlockHeight := message.SignedChunkRange.FirstBlockHeight()
	lastAvailableBlockHeight := message.SignedChunkRange.LastBlockHeight()
	b.reporting.Info("Received block sync response",
		log.Stringable("sender", message.Sender),
		log.Stringable("first-available-block-height", firstAvailableBlockHeight),
		log.Stringable("last-available-block-height", lastAvailableBlockHeight))
	for _, blockPair := range message.BlockPairs {
		_, err := b.storage.CommitBlock(&services.CommitBlockInput{blockPair})

		if err != nil {
			b.reporting.Error("Failed to commit block received via sync", log.Error(err))
		}
	}
}

func (b *BlockSync) PetitionerHandleBlockAvailabilityResponse(message *gossipmessages.BlockAvailabilityResponseMessage) error {
	b.reporting.Info("Received block availability response", log.Stringable("sender", message.Sender))

	senderPublicKey := message.Sender.SenderPublicKey()

	lastCommittedBlockHeight := b.storage.LastCommittedBlockHeight()

	if lastCommittedBlockHeight >= message.SignedBatchRange.LastCommittedBlockHeight() {
		return errors.New("source has is behind petitioner") // stay in the same state
	}

	blockType := message.SignedBatchRange.BlockType()

	lastAvailableBlockHeight := lastCommittedBlockHeight + primitives.BlockHeight(b.config.BlockSyncBatchSize())
	firstAvailableBlockHeight := lastCommittedBlockHeight + 1

	request := &gossiptopics.BlockSyncRequestInput{
		RecipientPublicKey: senderPublicKey,
		Message: &gossipmessages.BlockSyncRequestMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: b.config.NodePublicKey(),
			}).Build(),
			SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                blockType,
				LastBlockHeight:          lastAvailableBlockHeight,
				FirstBlockHeight:         firstAvailableBlockHeight,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
			}).Build(),
		},
	}

	_, err := b.gossip.SendBlockSyncRequest(request)
	return err
}

func (b *BlockSync) SourceHandleBlockAvailabilityRequest(message *gossipmessages.BlockAvailabilityRequestMessage) {
	b.reporting.Info("Received block availability request", log.Stringable("sender", message.Sender))

	lastCommittedBlockHeight := b.storage.LastCommittedBlockHeight()

	if lastCommittedBlockHeight == 0 {
		return
	}

	if lastCommittedBlockHeight <= message.SignedBatchRange.LastCommittedBlockHeight() {
		return
	}

	firstAvailableBlockHeight := primitives.BlockHeight(1)
	blockType := message.SignedBatchRange.BlockType()

	response := &gossiptopics.BlockAvailabilityResponseInput{
		RecipientPublicKey: message.Sender.SenderPublicKey(),
		Message: &gossipmessages.BlockAvailabilityResponseMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: b.config.NodePublicKey(),
			}).Build(),
			SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                blockType,
				LastBlockHeight:          lastCommittedBlockHeight,
				FirstBlockHeight:         firstAvailableBlockHeight,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
			}).Build(),
		},
	}
	b.gossip.SendBlockAvailabilityResponse(response)
}

func (b *BlockSync) SourceHandleBlockSyncRequest(message *gossipmessages.BlockSyncRequestMessage) {
	b.reporting.Info("Received block sync request", log.Stringable("sender", message.Sender))
	senderPublicKey := message.Sender.SenderPublicKey()
	blockType := message.SignedChunkRange.BlockType()
	lastCommittedBlockHeight := b.storage.LastCommittedBlockHeight()
	firstRequestedBlockHeight := message.SignedChunkRange.FirstBlockHeight()
	lastRequestedBlockHeight := message.SignedChunkRange.LastBlockHeight()
	if firstRequestedBlockHeight-lastCommittedBlockHeight > primitives.BlockHeight(b.config.BlockSyncBatchSize()-1) {
		lastRequestedBlockHeight = firstRequestedBlockHeight + primitives.BlockHeight(b.config.BlockSyncBatchSize()-1)
	}
	blocks, firstAvailableBlockHeight, lastAvailableBlockHeight := b.storage.GetBlocks(firstRequestedBlockHeight, lastRequestedBlockHeight)
	b.reporting.Info("Sending blocks to another node via block sync",
		log.Stringable("recipient", senderPublicKey),
		log.Stringable("first-available-block-height", firstAvailableBlockHeight),
		log.Stringable("last-available-block-height", lastAvailableBlockHeight))
	response := &gossiptopics.BlockSyncResponseInput{
		RecipientPublicKey: senderPublicKey,
		Message: &gossipmessages.BlockSyncResponseMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: b.config.NodePublicKey(),
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
	b.gossip.SendBlockSyncResponse(response)
}
