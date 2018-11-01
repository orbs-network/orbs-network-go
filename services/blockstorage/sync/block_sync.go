package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"time"
)

// this is coupled to gossip because the entire service is (block storage)
// nothing to gain right now in decoupling just the sync
type syncState interface {
	name() string
	String() string
	processState(ctx context.Context) syncState
	blockCommitted()
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
	LastCommittedBlockHeight() primitives.BlockHeight
	CommitBlock(ctx context.Context, input *services.CommitBlockInput) (*services.CommitBlockOutput, error)
	ValidateBlockForCommit(ctx context.Context, input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error)
	UpdateConsensusAlgosAboutLatestCommittedBlock(ctx context.Context)
}

type blockSyncConduit struct {
	Responses chan *gossipmessages.BlockAvailabilityResponseMessage
	Blocks    chan *gossipmessages.BlockSyncResponseMessage
}

type BlockSync struct {
	logger       log.BasicLogger
	sf           *stateFactory
	gossip       gossiptopics.BlockSync
	storage      BlockSyncStorage
	config       blockSyncConfig
	currentState syncState
	C            *blockSyncConduit
}

func NewBlockSync(ctx context.Context, config blockSyncConfig, gossip gossiptopics.BlockSync, storage BlockSyncStorage, logger log.BasicLogger) *BlockSync {
	conduit := &blockSyncConduit{
		Responses: make(chan *gossipmessages.BlockAvailabilityResponseMessage),
		Blocks:    make(chan *gossipmessages.BlockSyncResponseMessage),
	}

	bs := &BlockSync{
		logger:  logger.WithTags(log.String("flow", "block-sync")),
		sf:      NewStateFactory(config, gossip, storage, conduit, logger),
		gossip:  gossip,
		storage: storage,
		config:  config,
		C:       conduit,
	}

	bs.logger.Info("block sync init",
		log.Stringable("no-commit-timeout", bs.config.BlockSyncNoCommitInterval()),
		log.Stringable("collect-responses-timeout", bs.config.BlockSyncCollectResponseTimeout()),
		log.Stringable("collect-chunks-timeout", bs.config.BlockSyncCollectChunksTimeout()),
		log.Uint32("batch-size", bs.config.BlockSyncBatchSize()))

	supervised.LongLived(ctx, logger, func() {
		bs.syncLoop(ctx)
	})

	return bs
}

func (bs *BlockSync) syncLoop(ctx context.Context) {
	for bs.currentState = bs.sf.CreateCollectingAvailabilityResponseState(); bs.currentState != nil; {
		bs.logger.Info("state transitioning", log.Stringable("current-state", bs.currentState))
		// TODO add metrics
		bs.currentState = bs.currentState.processState(ctx)
	}
}

func (bs *BlockSync) HandleBlockCommitted() {
	if bs.currentState != nil {
		bs.currentState.blockCommitted()
	}
}

func (bs *BlockSync) HandleBlockAvailabilityResponse(ctx context.Context, input *gossiptopics.BlockAvailabilityResponseInput) (*gossiptopics.EmptyOutput, error) {
	if bs.currentState != nil {
		bs.currentState.gotAvailabilityResponse(input.Message)
	}
	return nil, nil
}

func (bs *BlockSync) HandleBlockSyncResponse(ctx context.Context, input *gossiptopics.BlockSyncResponseInput) (*gossiptopics.EmptyOutput, error) {
	if bs.currentState != nil {
		bs.currentState.gotBlocks(input.Message)
	}
	return nil, nil
}
