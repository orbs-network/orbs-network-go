package blockstorage

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
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
	BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES blockSyncState = 1
	BLOCK_SYNC_PETITIONER_WAITING_FOR_CHUNK                 blockSyncState = 2
)

type BlockSyncStorage interface {
	GetBlocks(first primitives.BlockHeight, last primitives.BlockHeight) (blocks []*protocol.BlockPairContainer, firstAvailableBlockHeight primitives.BlockHeight, lastAvailableBlockHeight primitives.BlockHeight)
	LastCommittedBlockHeight() primitives.BlockHeight
	CommitBlock(input *services.CommitBlockInput) (*services.CommitBlockOutput, error)
	ValidateBlockForCommit(input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error)
	UpdateConsensusAlgosAboutLatestCommittedBlock()
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
	defer func() {
		if r := recover(); r != nil {
			// TODO: in production we need to restart our long running goroutine (decide on supervision mechanism)
			b.reporting.Error("panic in BlockSync.mainLoop long running goroutine", log.String("panic", fmt.Sprintf("%v", r)))
		}
	}()

	state := BLOCK_SYNC_STATE_IDLE
	var event interface{}
	var blockAvailabilityResponses []*gossipmessages.BlockAvailabilityResponseMessage

	event = startSyncEvent{}

	startSyncTimer := synchronization.TempUntilJonathanTimer(b.config.BlockSyncInterval(), func() {
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

func (b *BlockSync) transitionState(currentState blockSyncState, event interface{}, availabilityResponses []*gossipmessages.BlockAvailabilityResponseMessage, startSyncTimer synchronization.TempUntilJonathanTrigger) (blockSyncState, []*gossipmessages.BlockAvailabilityResponseMessage) {
	// Ignore start sync because collecting availability responses has its own timer
	if _, ok := event.(startSyncEvent); ok && currentState != BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES {
		b.storage.UpdateConsensusAlgosAboutLatestCommittedBlock()

		err := b.petitionerBroadcastBlockAvailabilityRequest()

		if err != nil {
			b.reporting.Info("failed to broadcast block availability request", log.Error(err))
		} else {
			availabilityResponses = []*gossipmessages.BlockAvailabilityResponseMessage{}
			currentState = BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES
			startSyncTimer.Reset(b.config.BlockSyncInterval())

			time.AfterFunc(b.config.BlockSyncCollectResponseTimeout(), func() {
				b.events <- collectingAvailabilityFinishedEvent{}
			})
		}
	}

	if msg, ok := event.(*gossipmessages.BlockAvailabilityRequestMessage); ok {
		if err := b.sourceHandleBlockAvailabilityRequest(msg); err != nil {
			b.reporting.Info("failed to respond to block availability request", log.Error(err))
		}
	}

	if msg, ok := event.(*gossipmessages.BlockSyncRequestMessage); ok {
		if err := b.sourceHandleBlockSyncRequest(msg); err != nil {
			b.reporting.Info("failed to respond to block sync request", log.Error(err))
		}
	}

	switch currentState {
	case BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES:
		if msg, ok := event.(*gossipmessages.BlockAvailabilityResponseMessage); ok {
			availabilityResponses = append(availabilityResponses, msg)
			break
		}

		if _, ok := event.(collectingAvailabilityFinishedEvent); ok {
			count := len(availabilityResponses)

			if count == 0 {
				currentState = BLOCK_SYNC_STATE_IDLE
				break
			}

			b.reporting.Info("collected block availability responses", log.Int("num-responses", count))

			// TODO in the future we might want to have a more sophisticated select function than that
			syncSource := availabilityResponses[rand.Intn(count)]
			syncSourceKey := syncSource.Sender.SenderPublicKey()

			err := b.petitionerSendBlockSyncRequest(gossipmessages.BLOCK_TYPE_BLOCK_PAIR, syncSourceKey)
			if err != nil {
				b.reporting.Info("could not request block chunk from source", log.Error(err), log.Stringable("source", syncSource.Sender))
				currentState = BLOCK_SYNC_STATE_IDLE
			} else {
				b.reporting.Info("requested block chunk from source", log.Stringable("source", syncSource.Sender))
				currentState = BLOCK_SYNC_PETITIONER_WAITING_FOR_CHUNK

				startSyncTimer.Reset(b.config.BlockSyncCollectChunksTimeout())
			}
		}
	case BLOCK_SYNC_PETITIONER_WAITING_FOR_CHUNK:
		if msg, ok := event.(*gossipmessages.BlockSyncResponseMessage); ok {
			b.petitionerHandleBlockSyncResponse(msg)
			currentState = BLOCK_SYNC_STATE_IDLE
			startSyncTimer.Reset(0) // Fire immediately to sync next batch
		}
	}

	return currentState, availabilityResponses
}

func (b *BlockSync) petitionerBroadcastBlockAvailabilityRequest() error {
	lastCommittedBlockHeight := b.storage.LastCommittedBlockHeight()
	firstBlockHeight := lastCommittedBlockHeight + 1
	lastBlockHeight := lastCommittedBlockHeight + primitives.BlockHeight(b.config.BlockSyncBatchSize())

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
			break
		}

		_, err = b.storage.CommitBlock(&services.CommitBlockInput{BlockPair: blockPair})

		if err != nil {
			b.reporting.Error("failed to commit block received via sync", log.Error(err))
			break
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

func (b *BlockSync) sourceHandleBlockAvailabilityRequest(message *gossipmessages.BlockAvailabilityRequestMessage) error {
	b.reporting.Info("received block availability request", log.Stringable("sender", message.Sender))

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

	b.reporting.Info("received block sync request",
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
	_, err := b.gossip.SendBlockSyncResponse(response)
	return err
}
