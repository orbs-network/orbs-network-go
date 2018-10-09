package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
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

type blockSync struct {
	logger       log.BasicLogger
	terminated   bool
	sf           *stateFactory
	gossip       gossiptopics.BlockSync
	storage      BlockSyncStorage
	config       blockSyncConfig
	currentState syncState
}

func NewBlockSync(ctx context.Context, config blockSyncConfig, gossip gossiptopics.BlockSync, storage BlockSyncStorage) *blockSync {
	logger := log.GetLogger(log.Source("block-sync"))
	bs := &blockSync{
		logger:     logger,
		terminated: false,
		sf:         NewStateFactory(config, gossip, storage, logger),
		gossip:     gossip,
		storage:    storage,
		config:     config,
	}

	go bs.syncLoop(ctx)
	return bs
}

func (bs *blockSync) syncLoop(ctx context.Context) {
	bs.logger.Info("starting block sync main loop")
	for bs.currentState = bs.sf.CreateIdleState(); bs.currentState != nil; {
		bs.logger.Info("state transitioning", log.String("current-state", bs.currentState.name()))
		bs.currentState = bs.currentState.processState(ctx)
	}

	bs.terminated = true
	bs.logger.Info("block sync main loop ended")
}

func (bs *blockSync) HandleBlockAvailabilityRequest(input *gossiptopics.BlockAvailabilityRequestInput) (*gossiptopics.EmptyOutput, error) {
	return nil, nil
}

func (bs *blockSync) HandleBlockAvailabilityResponse(input *gossiptopics.BlockAvailabilityResponseInput) (*gossiptopics.EmptyOutput, error) {
	if bs.currentState != nil {
		bs.currentState.gotAvailabilityResponse(input.Message)
	}
	return nil, nil
}

func (bs *blockSync) HandleBlockSyncRequest(input *gossiptopics.BlockSyncRequestInput) (*gossiptopics.EmptyOutput, error) {
	return nil, nil
}

func (bs *blockSync) HandleBlockSyncResponse(input *gossiptopics.BlockSyncResponseInput) (*gossiptopics.EmptyOutput, error) {
	if bs.currentState != nil {
		bs.currentState.gotBlocks(input.Message)
	}
	return nil, nil
}
