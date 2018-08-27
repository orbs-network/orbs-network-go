package blockstorage

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

type BlockSync struct {
	reporting log.BasicLogger

	config  Config
	storage services.BlockStorage
	gossip  gossiptopics.BlockSync
	Events  chan interface{}
}

func NewBlockSync(ctx context.Context, storage services.BlockStorage, gossip gossiptopics.BlockSync, config Config, reporting log.BasicLogger) *BlockSync {
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
	for {
		//isActive := false
		//var syncSource primitives.Ed25519PublicKey

		select {
		case <-ctx.Done():
			return
		case event := <-b.Events:
			switch event.(type) {
			case *gossiptopics.BlockAvailabilityResponseInput:
				input := event.(*gossiptopics.BlockAvailabilityResponseInput)

				b.reporting.Info("Received block availability response", log.Stringable("sender", input.Message.Sender))

				senderPublicKey := input.Message.Sender.SenderPublicKey()

				//if isActive && !syncSource.Equal(senderPublicKey) {
				//	continue
				//}

				lastCommittedBlockHeightOutput, err := b.storage.GetLastCommittedBlockHeight(&services.GetLastCommittedBlockHeightInput{})
				if err != nil {
					continue
				}

				lastCommittedBlockHeight := lastCommittedBlockHeightOutput.LastCommittedBlockHeight

				if lastCommittedBlockHeight >= input.Message.SignedRange.LastCommittedBlockHeight() {
					continue
				}

				//syncSource = senderPublicKey
				//isActive = true

				blockType := input.Message.SignedRange.BlockType()

				lastAvailableBlockHeight := lastCommittedBlockHeight + primitives.BlockHeight(b.config.BlockSyncBatchSize())
				firstAvailableBlockHeight := lastCommittedBlockHeight + 1

				request := &gossiptopics.BlockSyncRequestInput{
					RecipientPublicKey: senderPublicKey,
					Message: &gossipmessages.BlockSyncRequestMessage{
						Sender: (&gossipmessages.SenderSignatureBuilder{
							SenderPublicKey: b.config.NodePublicKey(),
						}).Build(),
						SignedRange: (&gossipmessages.BlockSyncRangeBuilder{
							BlockType:                 blockType,
							LastAvailableBlockHeight:  lastAvailableBlockHeight,
							FirstAvailableBlockHeight: firstAvailableBlockHeight,
							LastCommittedBlockHeight:  lastCommittedBlockHeight,
						}).Build(),
					},
				}

				b.gossip.SendBlockSyncRequest(request)
			}
		}
	}
}
