package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"sync"
	"time"
)

type syncState interface {
	name() string
	processState(ctx context.Context) syncState
	blockCommitted(blockHeight primitives.BlockHeight)
	gotAvailabilityResponse(message *gossipmessages.BlockAvailabilityResponseMessage)
	gotBlocks(message *gossipmessages.BlockSyncResponseMessage)
}

type blockSyncConfig interface {
	NodePublicKey() primitives.Ed25519PublicKey
	BlockSyncBatchSize() uint32
	BlockSyncNoCommitInterval() time.Duration
	BlockSyncCollectResponseTimeout() time.Duration
	BlockSyncCollectChunksTimeout() time.Duration
}

type BlockSyncStorage interface {
	GetBlocks(first primitives.BlockHeight, last primitives.BlockHeight) (blocks []*protocol.BlockPairContainer, firstAvailableBlockHeight primitives.BlockHeight, lastAvailableBlockHeight primitives.BlockHeight)
	LastCommittedBlockHeight() primitives.BlockHeight
	CommitBlock(input *services.CommitBlockInput) (*services.CommitBlockOutput, error)
	ValidateBlockForCommit(input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error)
	UpdateConsensusAlgosAboutLatestCommittedBlock()
}

type BlockSync struct {
	logger       log.BasicLogger
	terminated   bool
	sf           *stateFactory
	gossip       gossiptopics.BlockSync
	storage      BlockSyncStorage
	config       blockSyncConfig
	currentState syncState
	eventLock    *sync.Mutex
}

func NewBlockSync(ctx context.Context, config blockSyncConfig, gossip gossiptopics.BlockSync, storage BlockSyncStorage, logger log.BasicLogger) *BlockSync {
	bs := &BlockSync{
		logger:     logger.WithTags(log.String("flow", "block-sync")),
		terminated: false,
		sf:         NewStateFactory(config, gossip, storage, logger),
		gossip:     gossip,
		storage:    storage,
		config:     config,
		eventLock:  &sync.Mutex{},
	}

	bs.logger.Info("block sync init",
		log.Stringable("no-commit-timeout", bs.config.BlockSyncNoCommitInterval()),
		log.Stringable("collect-responses-timeout", bs.config.BlockSyncCollectResponseTimeout()),
		log.Stringable("collect-chunks-timeout", bs.config.BlockSyncCollectChunksTimeout()),
		log.Uint32("batch-size", bs.config.BlockSyncBatchSize()))

	go bs.syncLoop(ctx)
	return bs
}

func (bs *BlockSync) syncLoop(ctx context.Context) {
	meter := bs.logger.Meter("inter-sync-main-loop")
	for bs.currentState = bs.sf.CreateIdleState(); bs.currentState != nil; {
		bs.logger.Info("state transitioning", log.String("current-state", bs.currentState.name()))
		bs.currentState = bs.currentState.processState(ctx)
	}

	bs.terminated = true
	meter.Done()
}

func (bs *BlockSync) HandleBlockAvailabilityRequest(input *gossiptopics.BlockAvailabilityRequestInput) (*gossiptopics.EmptyOutput, error) {
	bs.eventLock.Lock()
	defer bs.eventLock.Unlock()
	return nil, nil
}

func (bs *BlockSync) HandleBlockAvailabilityResponse(input *gossiptopics.BlockAvailabilityResponseInput) (*gossiptopics.EmptyOutput, error) {
	bs.eventLock.Lock()
	defer bs.eventLock.Unlock()

	bs.logger.Info("received availability response", log.Stringable("node-source", input.Message.Sender))
	if bs.currentState != nil {
		bs.currentState.gotAvailabilityResponse(input.Message)
	}
	return nil, nil
}

func (bs *BlockSync) HandleBlockSyncRequest(input *gossiptopics.BlockSyncRequestInput) (*gossiptopics.EmptyOutput, error) {
	bs.eventLock.Lock()
	defer bs.eventLock.Unlock()
	return nil, nil
}

func (bs *BlockSync) HandleBlockSyncResponse(input *gossiptopics.BlockSyncResponseInput) (*gossiptopics.EmptyOutput, error) {
	bs.eventLock.Lock()
	defer bs.eventLock.Unlock()

	if bs.currentState != nil {
		bs.currentState.gotBlocks(input.Message)
	}
	return nil, nil
}
