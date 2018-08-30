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
	"math/rand"
	"time"
)

type blockSyncState int

const (
	BLOCK_SYNC_STATE_IDLE                                   blockSyncState = 0
	BLOCK_SYNC_STATE_START_SYNC                             blockSyncState = 1
	BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES blockSyncState = 2
	BLOCK_SYNC_PETITIONER_ASK_FOR_BLOCKS                    blockSyncState = 3
	BLOCK_SYNC_PETITIONER_WAITING_FOR_CHUNK                 blockSyncState = 4
)

type BlockSyncStorage interface {
	GetBlocks(first primitives.BlockHeight, last primitives.BlockHeight) (blocks []*protocol.BlockPairContainer, firstAvailableBlockHeight primitives.BlockHeight, lastAvailableBlockHeight primitives.BlockHeight)
	LastCommittedBlockHeight() primitives.BlockHeight
	CommitBlock(input *services.CommitBlockInput) (*services.CommitBlockOutput, error)
	ValidateBlockForCommit(input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error)
	UpdateConsensusAlgo()
}

type BlockSync struct {
	reporting log.BasicLogger

	config  Config
	storage BlockSyncStorage
	gossip  gossiptopics.BlockSync
	events  chan interface{}
}

func NewBlockSync(ctx context.Context, config Config, storage BlockSyncStorage, gossip gossiptopics.BlockSync, reporting log.BasicLogger) *BlockSync {
	blockSync := &BlockSync{
		reporting: reporting.For(log.String("flow", "block-sync")),
		config:    config,
		storage:   storage,
		gossip:    gossip,
		events:    make(chan interface{}),
	}

	go blockSync.mainLoop(ctx)

	return blockSync
}

func (b *BlockSync) mainLoop(ctx context.Context) {
	state := BLOCK_SYNC_STATE_START_SYNC
	var event interface{}
	var blockAvailabilityResponses []*gossipmessages.BlockAvailabilityResponseMessage
	updateState := make(chan blockSyncState)

	// TODO use better time patterns
	syncTrigger := time.AfterFunc(b.config.BlockSyncInterval(), func() {
		if state == BLOCK_SYNC_STATE_IDLE {
			updateState <- BLOCK_SYNC_STATE_START_SYNC
		}
	})

	// TODO use better time patterns
	requestBlocksTrigger := time.AfterFunc(b.config.BlockSyncCollectResponseTimeout(), func() {
		if state == BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES {
			updateState <- BLOCK_SYNC_PETITIONER_ASK_FOR_BLOCKS
		}
	})

	for {
		state, blockAvailabilityResponses = b.dispatchEvent(state, event, blockAvailabilityResponses)

		if state == BLOCK_SYNC_STATE_START_SYNC {
			syncTrigger.Stop()
			syncTrigger.Reset(b.config.BlockSyncInterval())

			b.storage.UpdateConsensusAlgo()

			blockAvailabilityResponses = []*gossipmessages.BlockAvailabilityResponseMessage{}

			err := b.petitionerBroadcastBlockAvailabilityRequest()

			if err != nil {
				b.reporting.Info("failed to broadcast block availability request")
			} else {
				state = BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES
				requestBlocksTrigger.Stop()
				requestBlocksTrigger.Reset(b.config.BlockSyncCollectResponseTimeout())
			}
		}

		if state == BLOCK_SYNC_PETITIONER_ASK_FOR_BLOCKS {
			requestBlocksTrigger.Stop()

			count := len(blockAvailabilityResponses)

			if count == 0 {
				state = BLOCK_SYNC_STATE_START_SYNC
				continue
			}

			b.reporting.Info("collected block availability responses", log.Int("num-responses", count))

			// TODO in the future we might want to have a more sophisticated select function than that
			syncSource := blockAvailabilityResponses[rand.Intn(count)]
			syncSourceKey := syncSource.Sender.SenderPublicKey()

			err := b.petitionerSendBlockSyncRequest(gossipmessages.BLOCK_TYPE_BLOCK_PAIR, syncSourceKey)
			if err != nil {
				b.reporting.Info("could not request block chunk from source", log.Error(err), log.Stringable("source", syncSource.Sender))
				state = BLOCK_SYNC_STATE_START_SYNC
				continue
			} else {
				b.reporting.Info("requested block chunk from source", log.Stringable("source", syncSource.Sender))
				state = BLOCK_SYNC_PETITIONER_WAITING_FOR_CHUNK
			}
		}

		select {
		case state = <-updateState:
			// Nullify the event because if we switched state, the event was processed already
			event = nil
			continue
		case <-ctx.Done():
			return
		case event = <-b.events:
			continue
		}
	}
}

func (b *BlockSync) dispatchEvent(state blockSyncState, event interface{}, blockAvailabilityResponses []*gossipmessages.BlockAvailabilityResponseMessage) (blockSyncState, []*gossipmessages.BlockAvailabilityResponseMessage) {
	switch event.(type) {
	case *gossipmessages.BlockAvailabilityRequestMessage:
		message := event.(*gossipmessages.BlockAvailabilityRequestMessage)
		if fromMe := message.Sender.SenderPublicKey().Equal(b.config.NodePublicKey()); !fromMe {
			b.sourceHandleBlockAvailabilityRequest(message)
		}
	case *gossipmessages.BlockAvailabilityResponseMessage:
		message := event.(*gossipmessages.BlockAvailabilityResponseMessage)

		if fromMe := message.Sender.SenderPublicKey().Equal(b.config.NodePublicKey()); !fromMe {
			err := b.petitionerHandleBlockAvailabilityResponse(message)

			if err != nil {
				b.reporting.Info("received bad block availability response", log.Error(err))
			} else {
				blockAvailabilityResponses = append(blockAvailabilityResponses, message)
			}
		}
	case *gossipmessages.BlockSyncRequestMessage:
		message := event.(*gossipmessages.BlockSyncRequestMessage)
		if fromMe := message.Sender.SenderPublicKey().Equal(b.config.NodePublicKey()); !fromMe {
			b.sourceHandleBlockSyncRequest(message)
			return BLOCK_SYNC_STATE_IDLE, blockAvailabilityResponses
		}
	case *gossipmessages.BlockSyncResponseMessage:
		message := event.(*gossipmessages.BlockSyncResponseMessage)
		if fromMe := message.Sender.SenderPublicKey().Equal(b.config.NodePublicKey()); !fromMe {
			b.petitionerHandleBlockSyncResponse(message)
			return BLOCK_SYNC_STATE_IDLE, blockAvailabilityResponses
		}
	}

	return state, blockAvailabilityResponses
}

func (b *BlockSync) petitionerBroadcastBlockAvailabilityRequest() error {
	lastCommittedBlockHeight := b.storage.LastCommittedBlockHeight()
	firstBlockHeight := lastCommittedBlockHeight + 1
	lastBlockHeight := b.storage.LastCommittedBlockHeight() + primitives.BlockHeight(b.config.BlockSyncBatchSize())

	b.reporting.Info("broadcast block availability request",
		log.Stringable("first-block-height", firstBlockHeight),
		log.Stringable("last-block-height", lastBlockHeight))

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

	_, err := b.gossip.BroadcastBlockAvailabilityRequest(input)
	return err
}

func (b *BlockSync) petitionerHandleBlockSyncResponse(message *gossipmessages.BlockSyncResponseMessage) {
	firstBlockHeight := message.SignedChunkRange.FirstBlockHeight()
	lastBlockHeight := message.SignedChunkRange.LastBlockHeight()

	b.reporting.Info("received block sync response",
		log.Stringable("sender", message.Sender),
		log.Stringable("first-block-height", firstBlockHeight),
		log.Stringable("last-block-height", lastBlockHeight))

	for _, blockPair := range message.BlockPairs {
		_, err := b.storage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{BlockPair: blockPair})

		if err != nil {
			b.reporting.Error("failed to commit block received via sync", log.Error(err))
		}

		_, err = b.storage.CommitBlock(&services.CommitBlockInput{BlockPair: blockPair})

		if err != nil {
			b.reporting.Error("failed to commit block received via sync", log.Error(err))
		}
	}
}

func (b *BlockSync) petitionerHandleBlockAvailabilityResponse(message *gossipmessages.BlockAvailabilityResponseMessage) error {
	b.reporting.Info("received block availability response", log.Stringable("sender", message.Sender))

	lastCommittedBlockHeight := b.storage.LastCommittedBlockHeight()

	if lastCommittedBlockHeight >= message.SignedBatchRange.LastCommittedBlockHeight() {
		return errors.New("source is behind petitioner") // stay in the same state
	}

	return nil
}

func (b *BlockSync) petitionerSendBlockSyncRequest(blockType gossipmessages.BlockType, senderPublicKey primitives.Ed25519PublicKey) error {
	lastCommittedBlockHeight := b.storage.LastCommittedBlockHeight()

	firstBlockHeight := lastCommittedBlockHeight + 1
	lastBlockHeight := lastCommittedBlockHeight + primitives.BlockHeight(b.config.BlockSyncBatchSize())

	request := &gossiptopics.BlockSyncRequestInput{
		RecipientPublicKey: senderPublicKey,
		Message: &gossipmessages.BlockSyncRequestMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: b.config.NodePublicKey(),
			}).Build(),
			SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                blockType,
				LastBlockHeight:          lastBlockHeight,
				FirstBlockHeight:         firstBlockHeight,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
			}).Build(),
		},
	}

	_, err := b.gossip.SendBlockSyncRequest(request)
	return err
}

func (b *BlockSync) sourceHandleBlockAvailabilityRequest(message *gossipmessages.BlockAvailabilityRequestMessage) {
	b.reporting.Info("received block availability request", log.Stringable("sender", message.Sender))

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

func (b *BlockSync) sourceHandleBlockSyncRequest(message *gossipmessages.BlockSyncRequestMessage) {
	b.reporting.Info("received block sync request", log.Stringable("sender", message.Sender))
	senderPublicKey := message.Sender.SenderPublicKey()
	blockType := message.SignedChunkRange.BlockType()
	lastCommittedBlockHeight := b.storage.LastCommittedBlockHeight()
	firstRequestedBlockHeight := message.SignedChunkRange.FirstBlockHeight()
	lastRequestedBlockHeight := message.SignedChunkRange.LastBlockHeight()
	if firstRequestedBlockHeight-lastCommittedBlockHeight > primitives.BlockHeight(b.config.BlockSyncBatchSize()-1) {
		lastRequestedBlockHeight = firstRequestedBlockHeight + primitives.BlockHeight(b.config.BlockSyncBatchSize()-1)
	}

	blocks, firstAvailableBlockHeight, lastAvailableBlockHeight := b.storage.GetBlocks(firstRequestedBlockHeight, lastRequestedBlockHeight)
	b.reporting.Info("sending blocks to another node via block sync",
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
