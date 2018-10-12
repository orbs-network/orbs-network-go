package blockstorage

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
)

type blockSyncState int

const (
	BLOCK_SYNC_STATE_IDLE                                   blockSyncState = 0
	BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES blockSyncState = 1
	BLOCK_SYNC_PETITIONER_WAITING_FOR_CHUNK                 blockSyncState = 2
)

var BlockSyncFlowLogTag = log.String("flow", "block-sync")

type BlockSyncStorage interface {
	GetBlocks(first primitives.BlockHeight, last primitives.BlockHeight) (blocks []*protocol.BlockPairContainer, firstAvailableBlockHeight primitives.BlockHeight, lastAvailableBlockHeight primitives.BlockHeight)
	LastCommittedBlockHeight() primitives.BlockHeight
	CommitBlock(input *services.CommitBlockInput) (*services.CommitBlockOutput, error)
	ValidateBlockForCommit(input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error)
	UpdateConsensusAlgosAboutLatestCommittedBlock()
}

type BlockSync struct {
	logger log.BasicLogger

	config  config.BlockStorageConfig
	storage BlockSyncStorage
	gossip  gossiptopics.BlockSync
	events  chan interface{}
}

func NewBlockSync(ctx context.Context, config config.BlockStorageConfig, storage BlockSyncStorage, gossip gossiptopics.BlockSync, logger log.BasicLogger) *BlockSync {
	blockSync := &BlockSync{
		logger:  logger.WithTags(BlockSyncFlowLogTag),
		config:  config,
		storage: storage,
		gossip:  gossip,
		events:  make(chan interface{}),
	}

	go blockSync.mainLoop(ctx)

	return blockSync
}

func (b *BlockSync) mainLoop(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			// TODO: in production we need to restart our long running goroutine (decide on supervision mechanism)
			b.logger.Error("panic in BlockSync.mainLoop long running goroutine", log.String("panic", fmt.Sprintf("%v", r)))
		}
	}()

	state := BLOCK_SYNC_STATE_IDLE
	var event interface{}
	var blockAvailabilityResponses []*gossipmessages.BlockAvailabilityResponseMessage

	event = startSyncEvent{}

	startSyncTimer := synchronization.NewTrigger(b.config.BlockSyncNoCommitInterval(), func() {
		b.events <- startSyncEvent{}
	})

	for {
		state, blockAvailabilityResponses = b.transitionState(state, event, blockAvailabilityResponses, startSyncTimer)

		select {
		case <-ctx.Done():
			return
		case event = <-b.events:
			continue
		}
	}
}

type startSyncEvent struct{}
type collectingAvailabilityFinishedEvent struct{}

func (b *BlockSync) transitionState(currentState blockSyncState, event interface{}, availabilityResponses []*gossipmessages.BlockAvailabilityResponseMessage, startSyncTimer synchronization.Trigger) (blockSyncState, []*gossipmessages.BlockAvailabilityResponseMessage) {
	// this is in actual the sync server code, needs to move to the new sync engine

	if msg, ok := event.(*gossipmessages.BlockAvailabilityRequestMessage); ok {
		if err := b.sourceHandleBlockAvailabilityRequest(msg); err != nil {
			b.logger.Info("failed to respond to block availability request", log.Error(err))
		}
	}

	if msg, ok := event.(*gossipmessages.BlockSyncRequestMessage); ok {
		if err := b.sourceHandleBlockSyncRequest(msg); err != nil {
			b.logger.Info("failed to respond to block sync request", log.Error(err))
		}
	}

	return currentState, availabilityResponses
}

func (b *BlockSync) sourceHandleBlockAvailabilityRequest(message *gossipmessages.BlockAvailabilityRequestMessage) error {
	b.logger.Info("received block availability request", log.Stringable("sender", message.Sender))

	lastCommittedBlockHeight := b.storage.LastCommittedBlockHeight()

	if lastCommittedBlockHeight <= message.SignedBatchRange.LastCommittedBlockHeight() {
		return nil
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
	_, err := b.gossip.SendBlockAvailabilityResponse(response)
	return err
}

func (b *BlockSync) sourceHandleBlockSyncRequest(message *gossipmessages.BlockSyncRequestMessage) error {
	senderPublicKey := message.Sender.SenderPublicKey()
	blockType := message.SignedChunkRange.BlockType()
	firstRequestedBlockHeight := message.SignedChunkRange.FirstBlockHeight()
	lastRequestedBlockHeight := message.SignedChunkRange.LastBlockHeight()
	lastCommittedBlockHeight := b.storage.LastCommittedBlockHeight()

	b.logger.Info("received block sync request",
		log.Stringable("sender", message.Sender),
		log.Stringable("first-requested-block-height", firstRequestedBlockHeight),
		log.Stringable("last-requested-block-height", lastRequestedBlockHeight),
		log.Stringable("last-committed-block-height", lastCommittedBlockHeight))

	if lastCommittedBlockHeight <= firstRequestedBlockHeight {
		return errors.New("firstBlockHeight is greater or equal to lastCommittedBlockHeight")
	}

	if firstRequestedBlockHeight-lastCommittedBlockHeight > primitives.BlockHeight(b.config.BlockSyncBatchSize()-1) {
		lastRequestedBlockHeight = firstRequestedBlockHeight + primitives.BlockHeight(b.config.BlockSyncBatchSize()-1)
	}

	blocks, firstAvailableBlockHeight, lastAvailableBlockHeight := b.storage.GetBlocks(firstRequestedBlockHeight, lastRequestedBlockHeight)

	b.logger.Info("sending blocks to another node via block sync",
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
	_, err := b.gossip.SendBlockSyncResponse(response)
	return err
}
