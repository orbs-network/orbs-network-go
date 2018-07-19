package leanhelix

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

type Config interface {
	NetworkSize(asOfBlock uint64) uint32
	NodePublicKey() primitives.Ed25519Pkey
}

type service struct {
	gossip                   gossiptopics.LeanHelix
	blockStorage             services.BlockStorage
	transactionPool          services.TransactionPool
	consensusContext         services.ConsensusContext
	reporting                instrumentation.Reporting
	loopControl              instrumentation.LoopControl
	config                   Config
	lastCommittedBlockHeight primitives.BlockHeight
	blocksForRounds          map[primitives.BlockHeight]*protocol.BlockPairContainer
	votesForActiveRound      chan bool
}

func NewLeanHelixConsensusAlgo(
	gossip gossiptopics.LeanHelix,
	blockStorage services.BlockStorage,
	transactionPool services.TransactionPool,
	consensusContext services.ConsensusContext,
	events instrumentation.Reporting,
	loopControl instrumentation.LoopControl,
	config Config,
	isLeader bool,
) services.ConsensusAlgoLeanHelix {

	s := &service{
		gossip:           gossip,
		blockStorage:     blockStorage,
		transactionPool:  transactionPool,
		consensusContext: consensusContext,
		reporting:        events,
		loopControl:      loopControl,
		config:           config,
		lastCommittedBlockHeight: 0, // TODO: improve
		blocksForRounds:          make(map[primitives.BlockHeight]*protocol.BlockPairContainer),
	}

	gossip.RegisterLeanHelixHandler(s)
	if isLeader {
		go s.buildBlocksEventLoop()
	}
	return s
}

func (s *service) OnNewConsensusRound(input *services.OnNewConsensusRoundInput) (*services.OnNewConsensusRoundOutput, error) {
	panic("Not implemented")
}

func (s *service) HandleTransactionsBlock(input *handlers.HandleTransactionsBlockInput) (*handlers.HandleTransactionsBlockOutput, error) {
	panic("Not implemented")
}

func (s *service) HandleResultsBlock(input *handlers.HandleResultsBlockInput) (*handlers.HandleResultsBlockOutput, error) {
	panic("Not implemented")
}

// runs only on non-leaders
func (s *service) HandleLeanHelixPrePrepare(input *gossiptopics.LeanHelixPrePrepareInput) (*gossiptopics.EmptyOutput, error) {
	blockHeight := input.Message.BlockPair.TransactionsBlock.Header.BlockHeight()
	s.blocksForRounds[blockHeight] = input.Message.BlockPair
	_, err := s.gossip.SendLeanHelixPrepare(&gossiptopics.LeanHelixPrepareInput{})
	return &gossiptopics.EmptyOutput{}, err
}

// runs only on leader
func (s *service) HandleLeanHelixPrepare(input *gossiptopics.LeanHelixPrepareInput) (*gossiptopics.EmptyOutput, error) {
	// TODO: we assume we only get votes for the active round, in the real world we can't assume this,
	// TODO:  but here since we don't move to the next round unless everybody voted, it's ok
	if s.votesForActiveRound == nil {
		panic("received vote while not collecting votes")
	}
	s.votesForActiveRound <- true
	return &gossiptopics.EmptyOutput{}, nil
}

// runs only on non-leaders
func (s *service) HandleLeanHelixCommit(input *gossiptopics.LeanHelixCommitInput) (*gossiptopics.EmptyOutput, error) {
	s.commitBlockAndMoveToNextRound()
	return &gossiptopics.EmptyOutput{}, nil
}

func (s *service) HandleLeanHelixViewChange(input *gossiptopics.LeanHelixViewChangeInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}

func (s *service) HandleLeanHelixNewView(input *gossiptopics.LeanHelixNewViewInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}

func (s *service) buildBlocksEventLoop() {

	s.loopControl.NewLoop("consensus_round", func() {

		// see if we need to propose a new block
		if s.blocksForRounds[s.lastCommittedBlockHeight+1] == nil {
			proposedBlock, err := s.proposeNextBlock()
			if err != nil {
				s.reporting.Error(err)
			}
			s.blocksForRounds[s.lastCommittedBlockHeight+1] = proposedBlock
		}

		// validate the current proposed block
		if s.blocksForRounds[s.lastCommittedBlockHeight+1] != nil {
			valid, err := s.collectVotesForBlock(s.blocksForRounds[s.lastCommittedBlockHeight+1])
			if err != nil {
				s.reporting.Error(err)
			}

			// commit the block if validated
			if valid {
				s.commitBlockAndMoveToNextRound()
				s.gossip.SendLeanHelixCommit(&gossiptopics.LeanHelixCommitInput{})
			}
		}

	})
}

func (s *service) proposeNextBlock() (*protocol.BlockPairContainer, error) {
	proposedTransactions, err := s.transactionPool.GetTransactionsForOrdering(&services.GetTransactionsForOrderingInput{
		MaxNumberOfTransactions: 1,
	})
	if err != nil {
		return nil, err
	}
	blockPair := &protocol.BlockPairContainer{
		TransactionsBlock: &protocol.TransactionsBlockContainer{
			Header: (&protocol.TransactionsBlockHeaderBuilder{
				ProtocolVersion: blockstorage.ProtocolVersion,
				BlockHeight:     primitives.BlockHeight(s.lastCommittedBlockHeight + 1),
			}).Build(),
			SignedTransactions: proposedTransactions.SignedTransactions,
		},
	}
	return blockPair, nil
}

func (s *service) collectVotesForBlock(blockPair *protocol.BlockPairContainer) (bool, error) {
	s.votesForActiveRound = make(chan bool)
	defer func() {
		close(s.votesForActiveRound)
		s.votesForActiveRound = nil
	}()

	_, err := s.gossip.SendLeanHelixPrePrepare(&gossiptopics.LeanHelixPrePrepareInput{
		Message: &gossipmessages.LeanHelixPrePrepareMessage{
			BlockPair: blockPair,
		},
	})
	if err != nil {
		return false, nil
	}

	gotConsensus := true
	// asking for votes from everybody except ourselves
	for i := uint32(0); i < s.config.NetworkSize(0)-1; i++ {
		gotConsensus = gotConsensus && <-s.votesForActiveRound
	}

	return gotConsensus, nil
}

func (s *service) commitBlockAndMoveToNextRound() {
	blockPair, found := s.blocksForRounds[s.lastCommittedBlockHeight+1]
	if !found {
		panic(fmt.Sprintf("Node [%v] is trying to commit a block that wasn't prepared", s.config.NodePublicKey()))
	}
	s.blockStorage.CommitBlock(&services.CommitBlockInput{
		BlockPair: blockPair,
	})
	delete(s.blocksForRounds, s.lastCommittedBlockHeight+1)
	s.lastCommittedBlockHeight += 1
}
