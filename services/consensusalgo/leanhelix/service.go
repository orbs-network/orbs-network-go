package leanhelix

import (
	"context"
	"fmt"
	"github.com/orbs-network/lean-helix-go"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"sync"
	"time"
)

var LogTag = log.Service("consensus-algo-lean-helix")

type Config interface {
	NetworkSize(asOfBlock uint64) uint32
	NodePublicKey() primitives.Ed25519PublicKey
	ConstantConsensusLeader() primitives.Ed25519PublicKey
	ActiveConsensusAlgo() consensus.ConsensusAlgoType
}

// TODO Eventually remove all code except Init() which calls the external lean-helix-go submodule
func Init() {
	s := leanhelix.NewLeanHelix(nil, nil, nil) // TODO impl me
	fmt.Println(s)
}

type service struct {
	gossip           gossiptopics.LeanHelix
	blockStorage     services.BlockStorage
	transactionPool  services.TransactionPool
	consensusContext services.ConsensusContext
	logger           log.BasicLogger
	config           Config

	lastCommittedBlockHeight primitives.BlockHeight
	blocksForRounds          map[primitives.BlockHeight]*protocol.BlockPairContainer
	blocksForRoundsMutex     *sync.RWMutex
	votesForActiveRound      chan bool
}

func NewLeanHelixConsensusAlgo(
	ctx context.Context,
	gossip gossiptopics.LeanHelix,
	blockStorage services.BlockStorage,
	transactionPool services.TransactionPool,
	consensusContext services.ConsensusContext,
	logger log.BasicLogger,
	config Config,
) services.ConsensusAlgoLeanHelix {

	panic("Don't use this - will be replaced by lean-helix-go submodule")

	s := &service{
		gossip:           gossip,
		blockStorage:     blockStorage,
		transactionPool:  transactionPool,
		consensusContext: consensusContext,
		logger:           logger.WithTags(LogTag),
		config:           config,
		lastCommittedBlockHeight: 0, // TODO: improve startup
		blocksForRounds:          make(map[primitives.BlockHeight]*protocol.BlockPairContainer),
		blocksForRoundsMutex:     &sync.RWMutex{},
	}

	gossip.RegisterLeanHelixHandler(s)
	if config.ActiveConsensusAlgo() == consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX && config.ConstantConsensusLeader().Equal(config.NodePublicKey()) {
		supervised.LongLived(ctx, logger, func() {
			s.consensusRoundRunLoop(ctx)
		})
	}
	return s
}

func (s *service) HandleBlockConsensus(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {
	panic("Not implemented")
}

func (s *service) HandleLeanHelixPrePrepare(ctx context.Context, input *gossiptopics.LeanHelixPrePrepareInput) (*gossiptopics.EmptyOutput, error) {
	return nil, s.validatorVoteForNewBlockProposal(ctx, input.Message.BlockPair)
}

func (s *service) HandleLeanHelixPrepare(ctx context.Context, input *gossiptopics.LeanHelixPrepareInput) (*gossiptopics.EmptyOutput, error) {
	s.leaderAddVoteFromValidator()
	return nil, nil
}

func (s *service) HandleLeanHelixCommit(ctx context.Context, input *gossiptopics.LeanHelixCommitInput) (*gossiptopics.EmptyOutput, error) {
	s.validatorHandleCommit(ctx)
	return &gossiptopics.EmptyOutput{}, nil
}

func (s *service) HandleLeanHelixViewChange(ctx context.Context, input *gossiptopics.LeanHelixViewChangeInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}

func (s *service) HandleLeanHelixNewView(ctx context.Context, input *gossiptopics.LeanHelixNewViewInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}

// TODO: make this select on a cancelable context
func (s *service) consensusRoundRunLoop(ctx context.Context) {

	for {
		s.logger.Info("entered consensus round with last committed block height", log.BlockHeight(s.lastCommittedBlockHeight))

		// see if we need to propose a new block
		err := s.leaderProposeNextBlockIfNeeded(ctx)
		if err != nil {
			s.logger.Error("leader failed to propose next block", log.Error(err))
			continue
		}

		// validate the current proposed block
		s.blocksForRoundsMutex.RLock()
		activeBlock := s.blocksForRounds[s.lastCommittedBlockHeight+1]
		s.blocksForRoundsMutex.RUnlock()
		if activeBlock != nil {
			err := s.leaderCollectVotesForBlock(ctx, activeBlock)
			if err != nil {
				s.logger.Error("leader failed to collect votes for block", log.Error(err))
				time.Sleep(10 * time.Millisecond) // TODO: handle network failures with some time of exponential backoff
				continue
			}

			// commit the block since it's validated
			s.lastCommittedBlockHeight = s.commitBlockAndMoveToNextRound(ctx)
			s.gossip.SendLeanHelixCommit(ctx, &gossiptopics.LeanHelixCommitInput{})
		}

	}
}
