package leanhelix

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

func (s *service) leaderProposeNextBlockIfNeeded() error {
	nextBlockHeight := s.lastCommittedBlockHeight + 1

	s.blocksForRoundsMutex.RLock()
	nextBlock := s.blocksForRounds[nextBlockHeight]
	s.blocksForRoundsMutex.RUnlock()
	if nextBlock != nil {
		return nil
	}

	txOutput, err := s.consensusContext.RequestNewTransactionsBlock(&services.RequestNewTransactionsBlockInput{
		BlockHeight:             s.lastCommittedBlockHeight + 1,
		MaxBlockSizeKb:          0,
		MaxNumberOfTransactions: 0,
		PrevBlockHash:           nil,
	})
	if err != nil {
		return err
	}

	txBlock := txOutput.TransactionsBlock
	txBlock.BlockProof = (&protocol.TransactionsBlockProofBuilder{}).Build()

	rxOutput, err := s.consensusContext.RequestNewResultsBlock(&services.RequestNewResultsBlockInput{
		BlockHeight:       s.lastCommittedBlockHeight + 1,
		PrevBlockHash:     nil,
		TransactionsBlock: txBlock,
	})
	if err != nil {
		return err
	}

	rxBlock := rxOutput.ResultsBlock
	rxBlock.BlockProof = (&protocol.ResultsBlockProofBuilder{}).Build()

	proposedBlockPair := &protocol.BlockPairContainer{
		TransactionsBlock: txBlock,
		ResultsBlock:      rxBlock,
	}
	// TODO refactor to a read/write methods (use defer to unlock)
	s.blocksForRoundsMutex.Lock()
	s.blocksForRounds[nextBlockHeight] = proposedBlockPair
	s.blocksForRoundsMutex.Unlock()

	s.reporting.Info("Proposed block pair for height", instrumentation.BlockHeight(nextBlockHeight))

	return nil
}

func (s *service) leaderCollectVotesForBlock(blockPair *protocol.BlockPairContainer) error {
	s.votesForActiveRound = make(chan bool)
	defer func() {
		close(s.votesForActiveRound)
		s.votesForActiveRound = nil
	}()

	_, err := s.gossip.SendLeanHelixPrePrepare(&gossiptopics.LeanHelixPrePrepareInput{
		Message: &gossipmessages.LeanHelixPrePrepareMessage{
			SignedHeader: (&consensus.LeanHelixBlockRefBuilder{}).Build(),
			Sender:       (&consensus.LeanHelixSenderSignatureBuilder{}).Build(),
			BlockPair:    blockPair,
		},
	})
	if err != nil {
		return err
	}

	// asking for votes from everybody except ourselves
	numOfRequiredVotes := int(s.config.NetworkSize(0)) - 1
	for i := 0; i < numOfRequiredVotes; i++ {
		<-s.votesForActiveRound
	}

	s.reporting.Info("Got the required votes", instrumentation.Int("votes", numOfRequiredVotes))

	return nil
}

func (s *service) validatorVoteForNewBlockProposal(blockPair *protocol.BlockPairContainer) error {
	blockHeight := blockPair.TransactionsBlock.Header.BlockHeight()

	s.blocksForRoundsMutex.Lock()
	s.blocksForRounds[blockHeight] = blockPair
	s.blocksForRoundsMutex.Unlock()

	s.reporting.Info("Voting as validator for block of height %d", instrumentation.BlockHeight(blockHeight))
	_, err := s.gossip.SendLeanHelixPrepare(&gossiptopics.LeanHelixPrepareInput{})
	return err
}

func (s *service) leaderAddVoteFromValidator() {
	// TODO: we assume we only get votes for the active round, in the real world we can't assume this,
	// TODO:  but here since we don't move to the next round unless everybody voted, it's ok
	if s.votesForActiveRound == nil {
		panic("received vote while not collecting votes")
	}
	s.votesForActiveRound <- true
}

func (s *service) commitBlockAndMoveToNextRound() primitives.BlockHeight {
	blockHeight := s.lastCommittedBlockHeight + 1

	s.blocksForRoundsMutex.RLock()
	blockPair, found := s.blocksForRounds[blockHeight]
	s.blocksForRoundsMutex.RUnlock()

	if !found {
		errorMessage := "trying to commit a block that wasn't prepared"
		s.reporting.Error(errorMessage, instrumentation.BlockHeight(blockHeight))
		panic(fmt.Errorf(errorMessage))
	}

	s.blockStorage.CommitBlock(&services.CommitBlockInput{
		BlockPair: blockPair,
	})

	s.blocksForRoundsMutex.Lock()
	delete(s.blocksForRounds, blockHeight)
	s.blocksForRoundsMutex.Unlock()

	return blockHeight
}
