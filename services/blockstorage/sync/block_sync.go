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
	gotBlocks(source primitives.Ed25519PublicKey, blocks []*protocol.BlockPairContainer)
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
	logger     log.BasicLogger
	shouldStop bool
	sf         *stateFactory
	gossip     gossiptopics.BlockSync
	storage    BlockSyncStorage
	config     blockSyncConfig
}

func NewBlockSync(ctx context.Context, config blockSyncConfig, gossip gossiptopics.BlockSync, storage BlockSyncStorage) *blockSync {
	logger := log.GetLogger(log.Source("block-sync"))
	bs := &blockSync{
		logger:     logger,
		shouldStop: false,
		sf:         NewStateFactory(config, gossip, storage, logger),
		gossip:     gossip,
		storage:    storage,
		config:     config,
	}

	go bs.syncLoop(ctx)
	return bs
}

func (bs *blockSync) Shutdown() {
	bs.shouldStop = true
}

func (bs *blockSync) syncLoop(ctx context.Context) {
	bs.logger.Info("starting block sync main loop")
	for state := bs.sf.CreateIdleState(); state != nil && !bs.shouldStop; {
		bs.logger.Info("state transitioning", log.String("current-state", state.name()))
		state = state.processState(ctx)
	}

	bs.logger.Info("block sync main loop ended")
}
